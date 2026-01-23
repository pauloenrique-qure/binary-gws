package collector

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemStatus string

const (
	StatusOnline  SystemStatus = "online"
	StatusOffline SystemStatus = "offline"
)

type ComputeMetrics struct {
	CPU    *CPUMetrics    `json:"cpu,omitempty"`
	Memory *MemoryMetrics `json:"memory,omitempty"`
	Disk   *DiskMetrics   `json:"disk,omitempty"`
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

type Collector struct {
	mu               sync.RWMutex
	lastCompute      *ComputeMetrics
	lastComputeTime  time.Time
	computeInterval  time.Duration
}

func New(computeIntervalSeconds int) *Collector {
	return &Collector{
		computeInterval: time.Duration(computeIntervalSeconds) * time.Second,
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
	percentages, err := cpu.Percent(time.Second, false)
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
