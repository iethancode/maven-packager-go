// Package timing 聚合每次构建的单模块耗时，识别瓶颈模块。
package timing

import (
	"sort"
	"sync"
)

// ModuleTiming 单模块耗时。
type ModuleTiming struct {
	Module    string `json:"module"`
	ElapsedMs int64  `json:"elapsedMs"`
	Success   bool   `json:"success"`
}

// Summary 整体耗时摘要。
type Summary struct {
	Total       int64          `json:"total"`
	Modules     []ModuleTiming `json:"modules"`
	Bottleneck  string         `json:"bottleneck"`
	BottleneckMs int64         `json:"bottleneckMs"`
}

// Collector 线程安全地聚合模块耗时事件。
type Collector struct {
	mu      sync.Mutex
	items   map[string]ModuleTiming
	order   []string
}

func NewCollector() *Collector {
	return &Collector{items: map[string]ModuleTiming{}}
}

// Add 记录或更新一个模块的耗时（同名以最大值胜出，避免 reactor 多阶段覆盖成 0）。
func (c *Collector) Add(module string, elapsedMs int64, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.items[module]; ok {
		if elapsedMs > existing.ElapsedMs {
			existing.ElapsedMs = elapsedMs
		}
		existing.Success = success
		c.items[module] = existing
		return
	}
	c.items[module] = ModuleTiming{Module: module, ElapsedMs: elapsedMs, Success: success}
	c.order = append(c.order, module)
}

// Reset 清空已有记录（用于新一轮构建）。
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]ModuleTiming{}
	c.order = nil
}

// Snapshot 产出当前摘要，按耗时降序（便于瓶颈高亮）。
func (c *Collector) Snapshot() Summary {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := make([]ModuleTiming, 0, len(c.items))
	var total int64
	for _, m := range c.order {
		if it, ok := c.items[m]; ok {
			items = append(items, it)
			total += it.ElapsedMs
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ElapsedMs > items[j].ElapsedMs })
	s := Summary{Total: total, Modules: items}
	if len(items) > 0 {
		s.Bottleneck = items[0].Module
		s.BottleneckMs = items[0].ElapsedMs
	}
	return s
}
