package metrics

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Metrics 表示服务器的 CPU 和内存使用率（百分比，0-100）。
type Metrics struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

// Collector 采集服务器 CPU/内存使用率，内置 TTL 缓存。
type Collector struct {
	cacheTTL    time.Duration
	mu          sync.Mutex
	cached      Metrics
	lastCollect time.Time
	prevIdle    uint64
	prevTotal   uint64
}

// NewCollector 创建 metrics 采集器，默认 3 秒 TTL。
func NewCollector() *Collector {
	return &Collector{cacheTTL: 3 * time.Second}
}

// Collect 返回当前 CPU/内存使用率，缓存期内返回缓存值。
func (c *Collector) Collect() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastCollect) < c.cacheTTL {
		return c.cached
	}

	cpu := c.readCPU()
	mem := c.readMemory()
	c.cached = Metrics{CPU: cpu, Memory: mem}
	c.lastCollect = time.Now()
	return c.cached
}

func (c *Collector) readCPU() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0
	}

	var vals [7]uint64
	for i := 0; i < 7 && i+1 < len(fields); i++ {
		vals[i], _ = strconv.ParseUint(fields[i+1], 10, 64)
	}
	// user + nice + system + idle + iowait + irq + softirq
	idle := vals[3] + vals[4] // idle + iowait
	total := vals[0] + vals[1] + vals[2] + vals[3] + vals[4] + vals[5] + vals[6]

	if c.prevTotal == 0 {
		// 首次采样，记录基线，返回 0
		c.prevIdle = idle
		c.prevTotal = total
		return 0
	}

	dTotal := total - c.prevTotal
	dIdle := idle - c.prevIdle
	c.prevIdle = idle
	c.prevTotal = total

	if dTotal == 0 {
		return 0
	}
	return float64(dTotal-dIdle) / float64(dTotal) * 100
}

func (c *Collector) readMemory() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	var total, available uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			available = val
		}
	}
	if total == 0 {
		return 0
	}
	return float64(total-available) / float64(total) * 100
}
