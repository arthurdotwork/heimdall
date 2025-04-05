package heimdall

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type GatewayConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type EndpointConfig struct {
	Path           string              `yaml:"path"`
	Target         string              `yaml:"target"`
	Method         string              `yaml:"method"`
	Headers        map[string][]string `yaml:"headers"`
	AllowedHeaders []string            `yaml:"allowed_headers"`
}

type Config struct {
	Gateway   GatewayConfig    `yaml:"gateway"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

func LoadFromFile(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) WithDefaults() *Config {
	if c.Gateway.Port == 0 {
		c.Gateway.Port = 8080
	}

	if c.Gateway.ReadTimeout == 0 {
		c.Gateway.ReadTimeout = 5 * time.Second
	}

	if c.Gateway.WriteTimeout == 0 {
		c.Gateway.WriteTimeout = 5 * time.Second
	}

	if c.Gateway.ShutdownTimeout == 0 {
		c.Gateway.ShutdownTimeout = 10 * time.Second
	}

	return c
}
