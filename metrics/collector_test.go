package metrics

import (
	"testing"
	"time"
)

func TestCollect_ReturnsValidRange(t *testing.T) {
	c := NewCollector()
	m := c.Collect()
	if m.CPU < 0 || m.CPU > 100 {
		t.Errorf("CPU out of range: %f", m.CPU)
	}
	if m.Memory < 0 || m.Memory > 100 {
		t.Errorf("Memory out of range: %f", m.Memory)
	}
}

func TestCollect_CachesTTL(t *testing.T) {
	c := NewCollector()
	c.cacheTTL = 100 * time.Millisecond
	m1 := c.Collect()
	m2 := c.Collect()
	// 缓存期内应返回同一结果
	if m1.CPU != m2.CPU || m1.Memory != m2.Memory {
		t.Error("Expected cached result within TTL")
	}
	time.Sleep(150 * time.Millisecond)
	// 过期后应重新采集
	_ = c.Collect()
}
