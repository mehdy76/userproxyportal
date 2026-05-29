package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wisper/userproxyportal/internal/config"
)

var proxyEnvKeys = []string{
	"http_proxy", "HTTP_PROXY",
	"https_proxy", "HTTPS_PROXY",
	"no_proxy", "NO_PROXY",
}

// Apply configure les proxys GNOME vers le proxy local (sans root).
func Apply(cfg *config.Config) error {
	port := cfg.Proxy.GetLocalPort()
	return setGnomeManual("127.0.0.1", port)
}

// ApplyPrivileged écrit /etc/environment et installe le certificat (via pkexec).
func ApplyPrivileged(cfg *config.Config) error {
	localURL := cfg.Proxy.LocalProxyURL()
	envContent, err := buildEnvContent(localURL, cfg.Proxy.NoProxy)
	if err != nil {
		return err
	}

	certBlock := ""
	if cfg.Certificate.Path != "" {
		name := filepath.Base(cfg.Certificate.Path)
		ext := filepath.Ext(name)
		if !strings.EqualFold(ext, ".crt") {
			name = name[:len(name)-len(ext)] + ".crt"
		}
		certBlock = fmt.Sprintf(
			"cp '%s' '/usr/local/share/ca-certificates/%s' && update-ca-certificates &&\n",
			cfg.Certificate.Path, name,
		)
	}

	script := fmt.Sprintf(`set -e
%scat > /etc/environment << 'ENVEOF'
%s
ENVEOF`, certBlock, envContent)

	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("opérations privilégiées: %w\n%s", err, out)
	}
	return nil
}

// ClearAll supprime la configuration proxy GNOME et /etc/environment.
func ClearAll() error {
	if err := clearGnome(); err != nil {
		return err
	}
	return clearEnvPrivileged()
}

func buildEnvContent(localURL, noProxy string) (string, error) {
	var lines []string
	if f, err := os.Open("/etc/environment"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if !isProxyLine(line) && strings.TrimSpace(line) != "" {
				lines = append(lines, line)
			}
		}
		f.Close()
	}
	lines = append(lines,
		"http_proxy="+localURL,
		"HTTP_PROXY="+localURL,
		"https_proxy="+localURL,
		"HTTPS_PROXY="+localURL,
		"no_proxy="+noProxy,
		"NO_PROXY="+noProxy,
	)
	return strings.Join(lines, "\n"), nil
}

func isProxyLine(line string) bool {
	for _, key := range proxyEnvKeys {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			return true
		}
	}
	return false
}

func clearEnvPrivileged() error {
	var lines []string
	if f, err := os.Open("/etc/environment"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if !isProxyLine(line) && strings.TrimSpace(line) != "" {
				lines = append(lines, line)
			}
		}
		f.Close()
	}
	content := strings.Join(lines, "\n") + "\n"
	script := fmt.Sprintf("printf '%%s' '%s' > /etc/environment", strings.ReplaceAll(content, "'", "'\\''"))
	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("suppression proxy env: %w\n%s", err, out)
	}
	return nil
}

func gsettings(args ...string) error {
	out, err := exec.Command("gsettings", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gsettings %v: %w\n%s", args, err, out)
	}
	return nil
}

func setGnomeManual(host string, port int) error {
	portStr := strconv.Itoa(port)
	calls := [][]string{
		{"set", "org.gnome.system.proxy", "mode", "manual"},
		{"set", "org.gnome.system.proxy.http", "host", host},
		{"set", "org.gnome.system.proxy.http", "port", portStr},
		{"set", "org.gnome.system.proxy.http", "use-authentication", "false"},
		{"set", "org.gnome.system.proxy.https", "host", host},
		{"set", "org.gnome.system.proxy.https", "port", portStr},
		{"set", "org.gnome.system.proxy", "ignore-hosts", "['localhost', '127.0.0.0/8', '::1']"},
	}
	for _, args := range calls {
		if err := gsettings(args...); err != nil {
			return err
		}
	}
	return nil
}

func clearGnome() error {
	return gsettings("set", "org.gnome.system.proxy", "mode", "none")
}
