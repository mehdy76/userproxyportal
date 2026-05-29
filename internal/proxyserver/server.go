package proxyserver

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	nurl "net/url"
	"strings"
	"sync"
)

type Server struct {
	listenAddr   string
	upstreamAddr string

	mu       sync.RWMutex
	username string
	password string

	httpSrv *http.Server
}

func New(listenAddr, upstreamAddr string) *Server {
	return &Server{
		listenAddr:   listenAddr,
		upstreamAddr: upstreamAddr,
	}
}

func (s *Server) SetCredentials(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.username = username
	s.password = password
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
		http.Error(w, fmt.Sprintf("upstream: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

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
	upstream, err := net.Dial("tcp", s.upstreamAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("dial upstream: %v", err), http.StatusBadGateway)
		return
	}

	// Send CONNECT + auth to upstream proxy
	fmt.Fprintf(upstream, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n",
		r.Host, r.Host, s.authHeader())

	upstreamBuf := bufio.NewReader(upstream)
	upstreamResp, err := http.ReadResponse(upstreamBuf, r)
	if err != nil {
		upstream.Close()
		http.Error(w, fmt.Sprintf("upstream response: %v", err), http.StatusBadGateway)
		return
	}
	upstreamResp.Body.Close()

	if upstreamResp.StatusCode != http.StatusOK {
		upstream.Close()
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
