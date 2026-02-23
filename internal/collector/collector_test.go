package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetMonitoredProcesses(t *testing.T) {
	col := New(120)

	names := []string{"nginx", "python3", "redis-server"}
	col.SetMonitoredProcesses(names)

	col.mu.RLock()
	defer col.mu.RUnlock()

	if len(col.monitoredProcessNames) != len(names) {
		t.Errorf("expected %d monitored processes, got %d", len(names), len(col.monitoredProcessNames))
	}
	for i, name := range names {
		if col.monitoredProcessNames[i] != name {
			t.Errorf("expected monitoredProcessNames[%d]=%s, got %s", i, name, col.monitoredProcessNames[i])
		}
	}
}

func TestSetMonitoredProcessesOverwrites(t *testing.T) {
	col := New(120)

	col.SetMonitoredProcesses([]string{"nginx", "python3"})
	col.SetMonitoredProcesses([]string{"node"})

	col.mu.RLock()
	defer col.mu.RUnlock()

	if len(col.monitoredProcessNames) != 1 {
		t.Errorf("expected 1 monitored process after overwrite, got %d", len(col.monitoredProcessNames))
	}
	if col.monitoredProcessNames[0] != "node" {
		t.Errorf("expected monitoredProcessNames[0]=node, got %s", col.monitoredProcessNames[0])
	}
}

func TestCollectProcessNoMonitoredList(t *testing.T) {
	col := New(120)
	// No monitored processes set

	metrics := col.collectProcess()
	if metrics == nil {
		t.Fatal("collectProcess should return metrics even without monitored list")
	}

	if metrics.TotalCount <= 0 {
		t.Errorf("expected TotalCount > 0, got %d", metrics.TotalCount)
	}

	if len(metrics.MonitoredProcess) != 0 {
		t.Errorf("expected empty MonitoredProcess when no list is set, got %d entries", len(metrics.MonitoredProcess))
	}
}

func TestCollectProcessMonitoredListEmpty(t *testing.T) {
	col := New(120)
	col.SetMonitoredProcesses([]string{})

	metrics := col.collectProcess()
	if metrics == nil {
		t.Fatal("collectProcess should return metrics")
	}

	if len(metrics.MonitoredProcess) != 0 {
		t.Errorf("expected empty MonitoredProcess with empty list, got %d entries", len(metrics.MonitoredProcess))
	}
}

func TestCollectProcessFindsRunningProcess(t *testing.T) {
	col := New(120)

	execPath, err := os.Executable()
	if err != nil {
		t.Skip("could not determine executable name, skipping")
	}
	execName := filepath.Base(execPath)

	col.SetMonitoredProcesses([]string{execName})

	metrics := col.collectProcess()
	if metrics == nil {
		t.Fatal("collectProcess returned nil")
	}

	if len(metrics.MonitoredProcess) == 0 {
		t.Errorf("expected to find process %q in monitored list, but found none", execName)
	}

	for _, p := range metrics.MonitoredProcess {
		if p.Name == "" {
			t.Error("monitored process has empty name")
		}
		if p.PID <= 0 {
			t.Errorf("monitored process %q has invalid PID %d", p.Name, p.PID)
		}
		if p.Status == "" {
			t.Error("monitored process has empty status")
		}
	}
}

func TestCollectProcessUnknownProcess(t *testing.T) {
	col := New(120)
	col.SetMonitoredProcesses([]string{"this-process-definitely-does-not-exist-xyzzy"})

	metrics := col.collectProcess()
	if metrics == nil {
		t.Fatal("collectProcess returned nil")
	}

	if len(metrics.MonitoredProcess) != 0 {
		t.Errorf("expected 0 matches for non-existent process, got %d", len(metrics.MonitoredProcess))
	}
}

func TestCollectProcessCountsAreConsistent(t *testing.T) {
	col := New(120)

	metrics := col.collectProcess()
	if metrics == nil {
		t.Fatal("collectProcess returned nil")
	}

	if metrics.RunningCount < 0 {
		t.Errorf("RunningCount should not be negative, got %d", metrics.RunningCount)
	}
	if metrics.SleepingCount < 0 {
		t.Errorf("SleepingCount should not be negative, got %d", metrics.SleepingCount)
	}
	if metrics.RunningCount+metrics.SleepingCount > metrics.TotalCount {
		t.Errorf("RunningCount(%d) + SleepingCount(%d) > TotalCount(%d)",
			metrics.RunningCount, metrics.SleepingCount, metrics.TotalCount)
	}
}

func TestGetComputeMetricsCachesResult(t *testing.T) {
	col := New(120) // 120s cache

	first := col.GetComputeMetrics(false)
	second := col.GetComputeMetrics(false)

	if first != second {
		t.Error("expected same pointer from cache on second call")
	}
}

func TestGetComputeMetricsForceRefresh(t *testing.T) {
	col := New(120)

	first := col.GetComputeMetrics(false)
	second := col.GetComputeMetrics(true) // force refresh

	if first == second {
		t.Error("expected new metrics object after force refresh")
	}
}

func TestGetSystemStatus(t *testing.T) {
	col := New(120)
	if col.GetSystemStatus() != StatusOnline {
		t.Errorf("expected StatusOnline, got %s", col.GetSystemStatus())
	}
}

func TestTaskMetricsInitiallyNil(t *testing.T) {
	col := New(120)
	if col.GetTaskMetrics() != nil {
		t.Error("expected nil task metrics when no tasks have been recorded")
	}
}

func TestRecordTaskSuccess(t *testing.T) {
	col := New(120)
	col.RecordTaskSuccess("task-1")
	col.RecordTaskSuccess("task-2")

	metrics := col.GetTaskMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil task metrics after recording")
	}
	if metrics.TotalExecuted != 2 {
		t.Errorf("expected TotalExecuted=2, got %d", metrics.TotalExecuted)
	}
	if metrics.SuccessCount != 2 {
		t.Errorf("expected SuccessCount=2, got %d", metrics.SuccessCount)
	}
	if metrics.FailedCount != 0 {
		t.Errorf("expected FailedCount=0, got %d", metrics.FailedCount)
	}
}

func TestRecordTaskFailure(t *testing.T) {
	col := New(120)
	col.RecordTaskFailure("task-1", "connection refused")
	col.RecordTaskFailure("task-2", "timeout")

	metrics := col.GetTaskMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil task metrics")
	}
	if metrics.TotalExecuted != 2 {
		t.Errorf("expected TotalExecuted=2, got %d", metrics.TotalExecuted)
	}
	if metrics.FailedCount != 2 {
		t.Errorf("expected FailedCount=2, got %d", metrics.FailedCount)
	}
	if len(metrics.RecentFailures) != 2 {
		t.Errorf("expected 2 recent failures, got %d", len(metrics.RecentFailures))
	}
	if metrics.RecentFailures[0].TaskID != "task-1" {
		t.Errorf("expected first failure TaskID=task-1, got %s", metrics.RecentFailures[0].TaskID)
	}
}

func TestRecentFailuresCappedAt10(t *testing.T) {
	col := New(120)
	for i := 0; i < 15; i++ {
		col.RecordTaskFailure("task", "error")
	}

	metrics := col.GetTaskMetrics()
	if len(metrics.RecentFailures) != 10 {
		t.Errorf("expected RecentFailures capped at 10, got %d", len(metrics.RecentFailures))
	}
}
