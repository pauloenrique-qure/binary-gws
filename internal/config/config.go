package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	UUID      string    `yaml:"uuid"`
	ClientID  string    `yaml:"client_id"`
	SiteID    string    `yaml:"site_id"`
	APIURL    string    `yaml:"api_url"`
	Auth      Auth      `yaml:"auth"`
	Platform  Platform  `yaml:"platform"`
	Intervals Intervals `yaml:"intervals"`
	TLS       TLS       `yaml:"tls"`
}

type Auth struct {
	TokenCurrent string `yaml:"token_current"`
	TokenGrace   string `yaml:"token_grace"`
}

type Platform struct {
	PlatformOverride string `yaml:"platform_override"`
}

type Intervals struct {
	HeartbeatSeconds int `yaml:"heartbeat_seconds"`
	ComputeSeconds   int `yaml:"compute_seconds"`
}

type TLS struct {
	CABundlePath       string `yaml:"ca_bundle_path"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.setDefaults()
	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []string

	if c.UUID == "" {
		errs = append(errs, "uuid is required")
	}
	if c.ClientID == "" {
		errs = append(errs, "client_id is required")
	}
	if c.SiteID == "" {
		errs = append(errs, "site_id is required")
	}
	if c.APIURL == "" {
		errs = append(errs, "api_url is required")
	} else {
		parsedURL, err := url.Parse(c.APIURL)
		if err != nil {
			errs = append(errs, fmt.Sprintf("api_url is invalid: %v", err))
		} else if !strings.HasPrefix(strings.ToLower(parsedURL.Scheme), "http") {
			errs = append(errs, "api_url must be an HTTP(S) URL")
		}
	}
	if c.Auth.TokenCurrent == "" {
		errs = append(errs, "auth.token_current is required")
	}

	if c.Intervals.HeartbeatSeconds < 0 {
		errs = append(errs, "intervals.heartbeat_seconds cannot be negative")
	}
	if c.Intervals.ComputeSeconds < 0 {
		errs = append(errs, "intervals.compute_seconds cannot be negative")
	}

	if c.TLS.InsecureSkipVerify && c.TLS.CABundlePath != "" {
		errs = append(errs, "tls.insecure_skip_verify and tls.ca_bundle_path are mutually exclusive")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed: " + joinErrors(errs))
	}
	return nil
}

func (c *Config) setDefaults() {
	if c.Intervals.HeartbeatSeconds == 0 {
		c.Intervals.HeartbeatSeconds = 60
	}
	if c.Intervals.ComputeSeconds == 0 {
		c.Intervals.ComputeSeconds = 120
	}
}

func joinErrors(errs []string) string {
	return strings.Join(errs, "; ")
}
