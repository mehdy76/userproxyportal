package proxyserver

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	nurl "net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Couleurs ANSI (Linux uniquement)
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiGray   = "\033[90m"
)

type Server struct {
	listenAddr   string
	upstreamAddr string

	mu       sync.RWMutex
	username string
	password string

	debug   bool
	logger  *log.Logger
	httpSrv *http.Server
}

func New(listenAddr, upstreamAddr string) *Server {
	return &Server{
		listenAddr:   listenAddr,
		upstreamAddr: upstreamAddr,
		logger:       log.New(os.Stderr, "", 0),
	}
}

func (s *Server) SetCredentials(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.username = username
	s.password = password
}

func (s *Server) SetDebug(enabled bool) {
	s.debug = enabled
}

func (s *Server) Start() error {
	s.httpSrv = &http.Server{
		Addr:    s.listenAddr,
		Handler: s,
	}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) authHeader() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw := s.username + ":" + s.password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
}

func (s *Server) logRequest(method, target string, status int, duration time.Duration) {
	if !s.debug {
		return
	}

	ts := time.Now().Format("15:04:05")

	var statusColor string
	var note string
	switch {
	case status == 0:
		statusColor = ansiCyan
	case status == 407:
		statusColor = ansiRed
		note = "  ⚠  AUTH REFUSÉE — vérifiez vos credentials"
	case status == 403:
		statusColor = ansiRed
		note = "  ⚠  ACCÈS INTERDIT (filtrage)"
	case status < 300:
		statusColor = ansiGreen
	case status < 400:
		statusColor = ansiYellow
	default:
		statusColor = ansiRed
	}

	var statusStr string
	if status == 0 {
		statusStr = fmt.Sprintf("%stunnel%s", statusColor, ansiReset)
	} else {
		statusStr = fmt.Sprintf("%s%d%s", statusColor, status, ansiReset)
	}

	s.logger.Printf("%s%s%s  %-8s %s%-50s%s %s  %s%s%s%s",
		ansiGray, ts, ansiReset,
		method,
		ansiBold, target, ansiReset,
		statusStr,
		ansiGray, duration.Round(time.Millisecond), ansiReset,
		note,
	)
}

// CheckAuth teste l'authentification contre le proxy upstream sans démarrer le serveur.
func (s *Server) CheckAuth() error {
	s.mu.RLock()
	username := s.username
	s.mu.RUnlock()

	fmt.Fprintf(os.Stderr, "Test d'authentification...\n")
	fmt.Fprintf(os.Stderr, "  Upstream : %s\n", s.upstreamAddr)
	fmt.Fprintf(os.Stderr, "  Utilisateur : %s\n", username)

	upstream, err := net.DialTimeout("tcp", s.upstreamAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("impossible de joindre %s: %w", s.upstreamAddr, err)
	}
	defer upstream.Close()

	// Tente un CONNECT vers un hôte neutre
	testHost := "detectportal.firefox.com:443"
	fmt.Fprintf(upstream, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n",
		testHost, testHost, s.authHeader())

	upstream.SetDeadline(time.Now().Add(5 * time.Second))
	resp, err := http.ReadResponse(bufio.NewReader(upstream), nil)
	if err != nil {
		return fmt.Errorf("pas de réponse du proxy: %w", err)
	}
	resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintf(os.Stderr, "%s✓ Authentification OK%s\n", ansiGreen, ansiReset)
		return nil
	case http.StatusProxyAuthRequired:
		return fmt.Errorf("%s✗ Authentification refusée (407) — mauvais user/password%s", ansiRed, ansiReset)
	case http.StatusForbidden:
		return fmt.Errorf("%s✗ Accès interdit (403) — compte non autorisé%s", ansiRed, ansiReset)
	default:
		return fmt.Errorf("réponse inattendue: %s", resp.Status)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
	} else {
		s.handleHTTP(w, r)
	}
}

var hopByHopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailers", "Transfer-Encoding", "Upgrade",
}

func removeHopByHop(h http.Header) {
	for _, conn := range strings.Split(h.Get("Connection"), ",") {
		h.Del(strings.TrimSpace(conn))
	}
	for _, hdr := range hopByHopHeaders {
		h.Del(hdr)
	}
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	removeHopByHop(outReq.Header)
	outReq.Header.Set("Proxy-Authorization", s.authHeader())

	transport := &http.Transport{
		Proxy: func(*http.Request) (*nurl.URL, error) {
			return nurl.Parse("http://" + s.upstreamAddr)
		},
	}

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		s.logRequest(r.Method, r.URL.String(), http.StatusBadGateway, time.Since(start))
		http.Error(w, fmt.Sprintf("upstream: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	s.logRequest(r.Method, r.URL.String(), resp.StatusCode, time.Since(start))

	removeHopByHop(resp.Header)
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	upstream, err := net.Dial("tcp", s.upstreamAddr)
	if err != nil {
		s.logRequest("CONNECT", r.Host, http.StatusBadGateway, time.Since(start))
		http.Error(w, fmt.Sprintf("dial upstream: %v", err), http.StatusBadGateway)
		return
	}

	fmt.Fprintf(upstream, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n",
		r.Host, r.Host, s.authHeader())

	upstreamBuf := bufio.NewReader(upstream)
	upstreamResp, err := http.ReadResponse(upstreamBuf, r)
	if err != nil {
		upstream.Close()
		s.logRequest("CONNECT", r.Host, http.StatusBadGateway, time.Since(start))
		http.Error(w, fmt.Sprintf("upstream response: %v", err), http.StatusBadGateway)
		return
	}
	upstreamResp.Body.Close()

	if upstreamResp.StatusCode != http.StatusOK {
		upstream.Close()
		s.logRequest("CONNECT", r.Host, upstreamResp.StatusCode, time.Since(start))
		http.Error(w, fmt.Sprintf("upstream CONNECT: %s", upstreamResp.Status), http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		upstream.Close()
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		upstream.Close()
		return
	}

	// Tunnel établi — on logue avec status 0 (tunnel)
	s.logRequest("CONNECT", r.Host, 0, time.Since(start))

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(upstream, clientBuf)
		upstream.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, upstreamBuf)
		clientConn.Close()
	}()
	wg.Wait()
}
