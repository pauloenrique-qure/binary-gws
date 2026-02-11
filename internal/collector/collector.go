package collector

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemStatus string

const (
	StatusOnline  SystemStatus = "online"
	StatusOffline SystemStatus = "offline"
)

type ComputeMetrics struct {
	CPU         *CPUMetrics         `json:"cpu,omitempty"`
	Memory      *MemoryMetrics      `json:"memory,omitempty"`
	Disk        *DiskMetrics        `json:"disk,omitempty"`
	Temperature *TemperatureMetrics `json:"temperature,omitempty"`
	Process     *ProcessMetrics     `json:"process,omitempty"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
}

type MemoryMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type TemperatureMetrics struct {
	CPUCelsius float64          `json:"cpu_celsius,omitempty"`
	Sensors    []SensorReading  `json:"sensors,omitempty"`
}

type SensorReading struct {
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
}

type TaskMetrics struct {
	TotalExecuted  int64           `json:"total_executed"`
	FailedCount    int64           `json:"failed_count"`
	SuccessCount   int64           `json:"success_count"`
	LastFailure    string          `json:"last_failure,omitempty"`
	RecentFailures []TaskFailure   `json:"recent_failures,omitempty"`
}

type TaskFailure struct {
	TaskID    string `json:"task_id"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

type ProcessMetrics struct {
	TotalCount       int              `json:"total_count"`
	RunningCount     int              `json:"running_count"`
	SleepingCount    int              `json:"sleeping_count"`
	MonitoredProcess []ProcessInfo    `json:"monitored_processes,omitempty"`
}

type ProcessInfo struct {
	Name          string  `json:"name"`
	PID           int32   `json:"pid"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent,omitempty"`
	MemoryPercent float32 `json:"memory_percent,omitempty"`
	MemoryMB      uint64  `json:"memory_mb,omitempty"`
}

type Collector struct {
	mu               sync.RWMutex
	lastCompute      *ComputeMetrics
	lastComputeTime  time.Time
	computeInterval  time.Duration

	// Task tracking
	taskMu          sync.RWMutex
	totalExecuted   int64
	failedCount     int64
	successCount    int64
	lastFailureTime string
	recentFailures  []TaskFailure
	maxRecentFails  int

	// Process monitoring
	monitoredProcessNames []string
}

func New(computeIntervalSeconds int) *Collector {
	return &Collector{
		computeInterval: time.Duration(computeIntervalSeconds) * time.Second,
		maxRecentFails:  10, // Keep last 10 failures
		recentFailures:  make([]TaskFailure, 0, 10),
	}
}

func (c *Collector) GetSystemStatus() SystemStatus {
	return StatusOnline
}

func (c *Collector) GetComputeMetrics(force bool) *ComputeMetrics {
	now := time.Now()

	c.mu.RLock()
	if !force && c.lastCompute != nil && now.Sub(c.lastComputeTime) < c.computeInterval {
		cached := c.lastCompute
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	metrics := &ComputeMetrics{}
	hasAnyMetric := false

	if cpuMetric := c.collectCPU(); cpuMetric != nil {
		metrics.CPU = cpuMetric
		hasAnyMetric = true
	}

	if memMetric := c.collectMemory(); memMetric != nil {
		metrics.Memory = memMetric
		hasAnyMetric = true
	}

	if diskMetric := c.collectDisk(); diskMetric != nil {
		metrics.Disk = diskMetric
		hasAnyMetric = true
	}

	if tempMetric := c.collectTemperature(); tempMetric != nil {
		metrics.Temperature = tempMetric
		hasAnyMetric = true
	}

	if procMetric := c.collectProcess(); procMetric != nil {
		metrics.Process = procMetric
		hasAnyMetric = true
	}

	if !hasAnyMetric {
		return nil
	}

	c.mu.Lock()
	c.lastCompute = metrics
	c.lastComputeTime = now
	c.mu.Unlock()

	return metrics
}

func (c *Collector) collectCPU() *CPUMetrics {
	percentages, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil || len(percentages) == 0 {
		return nil
	}

	return &CPUMetrics{
		UsagePercent: percentages[0],
	}
}

func (c *Collector) collectMemory() *MemoryMetrics {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil
	}

	return &MemoryMetrics{
		TotalBytes:   vmStat.Total,
		UsedBytes:    vmStat.Used,
		UsagePercent: vmStat.UsedPercent,
	}
}

func (c *Collector) collectDisk() *DiskMetrics {
	var path string
	partitions, err := disk.Partitions(false)
	if err != nil || len(partitions) == 0 {
		if runtime.GOOS == "windows" {
			path = "C:"
		} else {
			path = "/"
		}
	} else {
		path = partitions[0].Mountpoint
	}

	usage, err := disk.Usage(path)
	if err != nil {
		return nil
	}

	return &DiskMetrics{
		TotalBytes:   usage.Total,
		UsedBytes:    usage.Used,
		UsagePercent: usage.UsedPercent,
	}
}

func (c *Collector) collectTemperature() *TemperatureMetrics {
	temps, err := host.SensorsTemperatures()
	if err != nil || len(temps) == 0 {
		return nil
	}

	metrics := &TemperatureMetrics{
		Sensors: make([]SensorReading, 0, len(temps)),
	}

	var cpuTemp float64
	cpuTempCount := 0

	for _, temp := range temps {
		// Add to sensors list
		metrics.Sensors = append(metrics.Sensors, SensorReading{
			Name:        temp.SensorKey,
			Temperature: temp.Temperature,
			Unit:        "celsius",
		})

		// Calculate average CPU temperature
		// Common CPU sensor names vary by platform
		if len(temp.SensorKey) >= 3 {
			key := temp.SensorKey[:3]
			if key == "cpu" || key == "CPU" || key == "cor" { // coretemp on Linux
				cpuTemp += temp.Temperature
				cpuTempCount++
			}
		}
	}

	if cpuTempCount > 0 {
		metrics.CPUCelsius = cpuTemp / float64(cpuTempCount)
	}

	return metrics
}

// RecordTaskSuccess records a successful task execution
func (c *Collector) RecordTaskSuccess(taskID string) {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()

	c.totalExecuted++
	c.successCount++
}

// RecordTaskFailure records a failed task execution
func (c *Collector) RecordTaskFailure(taskID, errorMsg string) {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()

	c.totalExecuted++
	c.failedCount++

	now := time.Now().UTC().Format(time.RFC3339)
	c.lastFailureTime = now

	failure := TaskFailure{
		TaskID:    taskID,
		Error:     errorMsg,
		Timestamp: now,
	}

	// Add to recent failures (keep last N)
	c.recentFailures = append(c.recentFailures, failure)
	if len(c.recentFailures) > c.maxRecentFails {
		c.recentFailures = c.recentFailures[1:]
	}
}

// GetTaskMetrics returns current task execution metrics
func (c *Collector) GetTaskMetrics() *TaskMetrics {
	c.taskMu.RLock()
	defer c.taskMu.RUnlock()

	if c.totalExecuted == 0 {
		return nil
	}

	metrics := &TaskMetrics{
		TotalExecuted: c.totalExecuted,
		FailedCount:   c.failedCount,
		SuccessCount:  c.successCount,
		LastFailure:   c.lastFailureTime,
	}

	// Copy recent failures
	if len(c.recentFailures) > 0 {
		metrics.RecentFailures = make([]TaskFailure, len(c.recentFailures))
		copy(metrics.RecentFailures, c.recentFailures)
	}

	return metrics
}

// SetMonitoredProcesses sets the list of process names to monitor
func (c *Collector) SetMonitoredProcesses(processNames []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.monitoredProcessNames = processNames
}

func (c *Collector) collectProcess() *ProcessMetrics {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	metrics := &ProcessMetrics{
		TotalCount:       len(procs),
		MonitoredProcess: make([]ProcessInfo, 0),
	}

	runningCount := 0
	sleepingCount := 0

	// Count process states and find monitored processes
	for _, p := range procs {
		status, err := p.Status()
		if err != nil {
			continue
		}

		// Count by status
		if len(status) > 0 {
			switch status[0] {
			case "R": // Running
				runningCount++
			case "S": // Sleeping
				sleepingCount++
			}
		}

		// Check if this is a monitored process
		if len(c.monitoredProcessNames) > 0 {
			name, err := p.Name()
			if err != nil {
				continue
			}

			// Check if this process name matches any monitored process
			for _, monitoredName := range c.monitoredProcessNames {
				if strings.Contains(strings.ToLower(name), strings.ToLower(monitoredName)) {
					info := ProcessInfo{
						Name:   name,
						PID:    p.Pid,
						Status: status[0],
					}

					// Try to get CPU and memory info (may fail on some systems)
					if cpuPercent, err := p.CPUPercent(); err == nil {
						info.CPUPercent = cpuPercent
					}

					if memInfo, err := p.MemoryInfo(); err == nil {
						info.MemoryMB = memInfo.RSS / 1024 / 1024
					}

					if memPercent, err := p.MemoryPercent(); err == nil {
						info.MemoryPercent = memPercent
					}

					metrics.MonitoredProcess = append(metrics.MonitoredProcess, info)
					break
				}
			}
		}
	}

	metrics.RunningCount = runningCount
	metrics.SleepingCount = sleepingCount

	return metrics
}
