package agent

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Metrics holds system resource usage information.
type Metrics struct {
	CPUPercent float64
	MemUsedMB  int64
	MemTotalMB int64
}

// CollectMetrics gathers current CPU and memory usage.
func CollectMetrics() Metrics {
	var m Metrics

	// Get CPU usage (non-blocking, returns last interval)
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		m.CPUPercent = cpuPercents[0]
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		m.MemUsedMB = int64(memInfo.Used / 1024 / 1024)
		m.MemTotalMB = int64(memInfo.Total / 1024 / 1024)
	}

	return m
}
