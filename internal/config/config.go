package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "/etc/userproxyportal/config.yaml"

type Config struct {
	Proxy       ProxyConfig       `yaml:"proxy"`
	Certificate CertificateConfig `yaml:"certificate"`
}

type ProxyConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	PACUrl    string `yaml:"pac_url"`
	NoProxy   string `yaml:"no_proxy"`
	LocalPort int    `yaml:"local_port"` // port du proxy local (défaut: 3128)
}

func (p *ProxyConfig) GetLocalPort() int {
	if p.LocalPort == 0 {
		return 3128
	}
	return p.LocalPort
}

func (p *ProxyConfig) LocalProxyURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.GetLocalPort())
}

type CertificateConfig struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// UpstreamURL builds the upstream proxy URL (without credentials).
func (c *Config) UpstreamURL() string {
	return fmt.Sprintf("http://%s:%d", c.Proxy.Host, c.Proxy.Port)
}
