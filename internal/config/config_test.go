package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid config",
			config: Config{
				UUID:     "test-uuid",
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "https://api.example.com",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
			},
			expectErr: false,
		},
		{
			name: "missing uuid",
			config: Config{
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "https://api.example.com",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
			},
			expectErr: true,
		},
		{
			name: "missing client_id",
			config: Config{
				UUID:   "test-uuid",
				SiteID: "test-site",
				APIURL: "https://api.example.com",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
			},
			expectErr: true,
		},
		{
			name: "missing token",
			config: Config{
				UUID:     "test-uuid",
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "https://api.example.com",
			},
			expectErr: true,
		},
		{
			name: "invalid url",
			config: Config{
				UUID:     "test-uuid",
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "not-a-url",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid fallback url",
			config: Config{
				UUID:            "test-uuid",
				ClientID:        "test-client",
				SiteID:          "test-site",
				APIURL:          "https://api.example.com",
				APIURLFallbacks: []string{"not-a-url"},
				Auth: Auth{
					TokenCurrent: "test-token",
				},
			},
			expectErr: true,
		},
		{
			name: "negative heartbeat interval",
			config: Config{
				UUID:     "test-uuid",
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "https://api.example.com",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
				Intervals: Intervals{
					HeartbeatSeconds: -1,
				},
			},
			expectErr: true,
		},
		{
			name: "conflicting tls config",
			config: Config{
				UUID:     "test-uuid",
				ClientID: "test-client",
				SiteID:   "test-site",
				APIURL:   "https://api.example.com",
				Auth: Auth{
					TokenCurrent: "test-token",
				},
				TLS: TLS{
					InsecureSkipVerify: true,
					CABundlePath:       "/path/to/ca.pem",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		UUID:     "test-uuid",
		ClientID: "test-client",
		SiteID:   "test-site",
		APIURL:   "https://api.example.com",
		Auth: Auth{
			TokenCurrent: "test-token",
		},
	}

	cfg.setDefaults()

	if cfg.Intervals.HeartbeatSeconds != 15 {
		t.Errorf("expected HeartbeatSeconds=15, got %d", cfg.Intervals.HeartbeatSeconds)
	}

	if cfg.Intervals.ComputeSeconds != 120 {
		t.Errorf("expected ComputeSeconds=120, got %d", cfg.Intervals.ComputeSeconds)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
uuid: test-gateway-123
client_id: test-client
site_id: test-site
api_url: https://api.example.com/heartbeat
api_url_fallbacks:
  - https://api-backup.example.com/heartbeat
auth:
  token_current: secret-token-123
  token_grace: old-token-456
intervals:
  heartbeat_seconds: 30
  compute_seconds: 60
tls:
  insecure_skip_verify: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.UUID != "test-gateway-123" {
		t.Errorf("expected UUID=test-gateway-123, got %s", cfg.UUID)
	}

	if cfg.Auth.TokenCurrent != "secret-token-123" {
		t.Errorf("expected token_current=secret-token-123, got %s", cfg.Auth.TokenCurrent)
	}

	if cfg.Intervals.HeartbeatSeconds != 30 {
		t.Errorf("expected HeartbeatSeconds=30, got %d", cfg.Intervals.HeartbeatSeconds)
	}
}

func TestMonitoredProcessesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("parses monitored_processes list", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config_procs.yaml")
		configContent := `
uuid: test-uuid
client_id: test-client
site_id: test-site
api_url: https://api.example.com
auth:
  token_current: secret-token
monitored_processes:
  - nginx
  - python3
  - redis-server
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if len(cfg.MonitoredProcesses) != 3 {
			t.Errorf("expected 3 monitored processes, got %d", len(cfg.MonitoredProcesses))
		}

		expected := []string{"nginx", "python3", "redis-server"}
		for i, name := range expected {
			if cfg.MonitoredProcesses[i] != name {
				t.Errorf("expected monitored_processes[%d]=%s, got %s", i, name, cfg.MonitoredProcesses[i])
			}
		}
	})

	t.Run("empty monitored_processes is valid", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config_no_procs.yaml")
		configContent := `
uuid: test-uuid
client_id: test-client
site_id: test-site
api_url: https://api.example.com
auth:
  token_current: secret-token
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("config without monitored_processes should be valid: %v", err)
		}

		if len(cfg.MonitoredProcesses) != 0 {
			t.Errorf("expected empty monitored processes, got %d", len(cfg.MonitoredProcesses))
		}
	})
}
