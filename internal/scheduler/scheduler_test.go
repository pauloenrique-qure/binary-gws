package scheduler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/binary-gws/agent/internal/collector"
	"github.com/binary-gws/agent/internal/logging"
	"github.com/binary-gws/agent/internal/platform"
)

func TestBuildPayload(t *testing.T) {
	platformInfo := &platform.Info{
		Platform: platform.PlatformUbuntu,
		OS:       "linux",
		Arch:     "amd64",
	}

	col := collector.New(120)
	logger := logging.New(logging.LevelInfo, nil, "test-uuid")

	sched := New(Config{
		UUID:             "test-gateway-123",
		ClientID:         "test-client",
		SiteID:           "test-site",
		Platform:         platformInfo,
		HeartbeatSeconds: 60,
		Collector:        col,
		Logger:           logger,
		Version:          "1.0.0",
		Commit:           "abc123",
		BuildDate:        "2024-01-01",
	})

	payload := sched.buildPayload()

	if payload.PayloadVersion != "1.0" {
		t.Errorf("expected PayloadVersion=1.0, got %s", payload.PayloadVersion)
	}

	if payload.UUID != "test-gateway-123" {
		t.Errorf("expected UUID=test-gateway-123, got %s", payload.UUID)
	}

	if payload.ClientID != "test-client" {
		t.Errorf("expected ClientID=test-client, got %s", payload.ClientID)
	}

	if payload.SiteID != "test-site" {
		t.Errorf("expected SiteID=test-site, got %s", payload.SiteID)
	}

	if payload.Stats.SystemStatus != collector.StatusOnline {
		t.Errorf("expected SystemStatus=online, got %s", payload.Stats.SystemStatus)
	}

	if payload.Additional.Metadata.Platform != platform.PlatformUbuntu {
		t.Errorf("expected Platform=ubuntu, got %s", payload.Additional.Metadata.Platform)
	}

	if payload.Additional.Metadata.AgentVersion != "1.0.0" {
		t.Errorf("expected AgentVersion=1.0.0, got %s", payload.Additional.Metadata.AgentVersion)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var rawPayload map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawPayload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	stats := rawPayload["stats"].(map[string]interface{})
	if _, hasCompute := stats["compute"]; hasCompute {
		t.Logf("compute metrics present (expected on first call with 1s delay)")
	}
}

func TestPayloadOmitsMissingMetrics(t *testing.T) {
	platformInfo := &platform.Info{
		Platform: platform.PlatformLinux,
	}

	col := collector.New(120)
	logger := logging.New(logging.LevelInfo, nil, "test-uuid")

	sched := New(Config{
		UUID:             "test-uuid",
		ClientID:         "client",
		SiteID:           "site",
		Platform:         platformInfo,
		HeartbeatSeconds: 60,
		Collector:        col,
		Logger:           logger,
	})

	time.Sleep(2 * time.Second)

	payload := sched.buildPayload()
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var rawPayload map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawPayload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	stats := rawPayload["stats"].(map[string]interface{})

	if _, hasSystemStatus := stats["system_status"]; !hasSystemStatus {
		t.Error("system_status should always be present")
	}
}

func TestDryRun(t *testing.T) {
	platformInfo := &platform.Info{
		Platform: platform.PlatformLinux,
	}

	col := collector.New(120)
	logger := logging.New(logging.LevelInfo, nil, "test-uuid")

	sched := New(Config{
		UUID:             "test-uuid",
		ClientID:         "client",
		SiteID:           "site",
		Platform:         platformInfo,
		HeartbeatSeconds: 60,
		Collector:        col,
		Logger:           logger,
	})

	ctx := context.Background()
	err := sched.SendOnce(ctx, true)
	if err != nil {
		t.Errorf("dry run should not fail: %v", err)
	}
}
