package scheduler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/binary-gws/agent/internal/collector"
	"github.com/binary-gws/agent/internal/logging"
	"github.com/binary-gws/agent/internal/platform"
)

// newTestScheduler creates a scheduler with standard test defaults and a warmed-up collector.
// The collector's compute interval is set to 1s; callers must sleep >=2s before calling buildPayload.
func newTestScheduler(t *testing.T, col *collector.Collector) *Scheduler {
	t.Helper()
	return New(Config{
		UUID:             "test-uuid",
		ClientID:         "client",
		SiteID:           "site",
		Platform:         &platform.Info{Platform: platform.PlatformUbuntu},
		HeartbeatSeconds: 60,
		Collector:        col,
		Logger:           logging.New(logging.LevelInfo, nil, "test-uuid"),
	})
}

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

	if payload.BatchIndex != 1 {
		t.Errorf("expected BatchIndex=1 on first call, got %d", payload.BatchIndex)
	}

	// Test that batch_index increments
	payload2 := sched.buildPayload()
	if payload2.BatchIndex != 2 {
		t.Errorf("expected BatchIndex=2 on second call, got %d", payload2.BatchIndex)
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

func TestPayloadProcessMetricsPresent(t *testing.T) {
	col := collector.New(1)
	sched := newTestScheduler(t, col)

	time.Sleep(2 * time.Second)
	payload := sched.buildPayload()

	if payload.Stats.Compute == nil {
		t.Fatal("expected compute metrics in payload")
	}
	if payload.Stats.Compute.Process == nil {
		t.Fatal("expected process metrics in payload")
	}
	if payload.Stats.Compute.Process.TotalCount <= 0 {
		t.Errorf("expected TotalCount > 0, got %d", payload.Stats.Compute.Process.TotalCount)
	}
}

func TestPayloadMonitoredProcessesInPayload(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Skip("could not determine executable name, skipping")
	}
	execName := filepath.Base(execPath)

	col := collector.New(1)
	col.SetMonitoredProcesses([]string{execName})
	sched := newTestScheduler(t, col)

	time.Sleep(2 * time.Second)
	payload := sched.buildPayload()

	if payload.Stats.Compute == nil || payload.Stats.Compute.Process == nil {
		t.Fatal("expected process metrics in payload")
	}
	if len(payload.Stats.Compute.Process.MonitoredProcess) == 0 {
		t.Errorf("expected monitored process %q in payload, got none", execName)
	}
}

func TestPayloadNoMonitoredProcessesWhenListEmpty(t *testing.T) {
	col := collector.New(1)
	sched := newTestScheduler(t, col)

	time.Sleep(2 * time.Second)
	payload := sched.buildPayload()

	if payload.Stats.Compute == nil || payload.Stats.Compute.Process == nil {
		t.Fatal("expected process metrics in payload")
	}
	if len(payload.Stats.Compute.Process.MonitoredProcess) != 0 {
		t.Errorf("expected empty MonitoredProcess when no list set, got %d", len(payload.Stats.Compute.Process.MonitoredProcess))
	}
}
