package collector

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type MetricsPayload struct {
	CPU     CPUMetrics     `json:"cpu"`
	Memory  MemoryMetrics  `json:"memory"`
	Disk    DiskMetrics    `json:"disk"`
	Network NetworkMetrics `json:"network"`
	System  SystemMetrics  `json:"system"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

type MemoryMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkMetrics struct {
	BytesRecvPerSec float64 `json:"bytes_recv_per_sec"`
	BytesSentPerSec float64 `json:"bytes_sent_per_sec"`
}

type SystemMetrics struct {
	UptimeSeconds uint64 `json:"uptime_seconds"`
	OSName        string `json:"os_name"`
}

type Collector struct {
	prevNetRecv uint64
	prevNetSent uint64
	prevTime    time.Time
}

// NewCollector initializes the metrics collector.
func NewCollector() *Collector {
	c := &Collector{
		prevTime: time.Now(),
	}
	// Initial read of net counters to set baseline
	if io, err := net.IOCounters(false); err == nil && len(io) > 0 {
		c.prevNetRecv = io[0].BytesRecv
		c.prevNetSent = io[0].BytesSent
	}
	return c
}

// Collect reads current system metrics and calculates delta values for network bandwidth.
func (c *Collector) Collect() (*MetricsPayload, error) {
	var payload MetricsPayload

	// 1. CPU Load
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		payload.CPU.UsagePercent = cpuPercents[0]
	}
	cores, err := cpu.Counts(true)
	if err == nil {
		payload.CPU.Cores = cores
	}

	// 2. Memory Usage
	vmem, err := mem.VirtualMemory()
	if err == nil {
		payload.Memory.TotalBytes = vmem.Total
		payload.Memory.UsedBytes = vmem.Used
		payload.Memory.FreeBytes = vmem.Free
		payload.Memory.UsagePercent = vmem.UsedPercent
	}

	// 3. Disk Usage (main root mount)
	diskUsage, err := disk.Usage("/")
	if err == nil {
		payload.Disk.TotalBytes = diskUsage.Total
		payload.Disk.UsedBytes = diskUsage.Used
		payload.Disk.FreeBytes = diskUsage.Free
		payload.Disk.UsagePercent = diskUsage.UsedPercent
	}

	// 4. Network Rate calculation
	now := time.Now()
	io, err := net.IOCounters(false)
	if err == nil && len(io) > 0 {
		duration := now.Sub(c.prevTime).Seconds()
		if duration > 0 {
			recvDiff := io[0].BytesRecv - c.prevNetRecv
			sentDiff := io[0].BytesSent - c.prevNetSent
			payload.Network.BytesRecvPerSec = float64(recvDiff) / duration
			payload.Network.BytesSentPerSec = float64(sentDiff) / duration
		}
		c.prevNetRecv = io[0].BytesRecv
		c.prevNetSent = io[0].BytesSent
	}
	c.prevTime = now

	// 5. System metrics (Uptime & OS pretty name)
	hinfo, err := host.Info()
	if err == nil {
		payload.System.UptimeSeconds = hinfo.Uptime
		payload.System.OSName = hinfo.OS + " " + hinfo.PlatformVersion
	}

	return &payload, nil
}
