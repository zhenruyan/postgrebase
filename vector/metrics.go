package vector

import (
	"runtime"
	"time"
)

// Metrics describes the single-node runtime metrics surfaced by the monitoring
// panel. Cluster-wide metrics are derived from the coordinator view.
type Metrics struct {
	NodeID            string    `json:"nodeId"`
	Mode              Mode      `json:"mode"`
	Online            bool      `json:"online"`
	UptimeSeconds     int64     `json:"uptimeSeconds"`
	Goroutines        int       `json:"goroutines"`
	MemAllocBytes     uint64    `json:"memAllocBytes"`
	MemSysBytes       uint64    `json:"memSysBytes"`
	HeapObjects       uint64    `json:"heapObjects"`
	GCCount           uint32    `json:"gcCount"`
	NumCPU            int       `json:"numCpu"`
	PendingEmbeddings int       `json:"pendingEmbeddings"`
	VectorEntries     int       `json:"vectorEntries"`
	CacheItems        int       `json:"cacheItems"`
	CacheBackend      string    `json:"cacheBackend"`
	EmbeddingModel    string    `json:"embeddingModel"`
	EmbeddingReady    bool      `json:"embeddingReady"`
	RedisEnabled      bool      `json:"redisEnabled"`
	Backend           string    `json:"backend"`
	DataDriver        string    `json:"dataDriver"`
	CollectedAt       time.Time `json:"collectedAt"`
}

// Metrics collects the current single-node runtime metrics.
func (m *Manager) Metrics() Metrics {
	m.mu.RLock()
	status := m.status
	entries := len(m.entries)
	m.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := int64(0)
	if !status.StartedAt.IsZero() {
		uptime = int64(time.Since(status.StartedAt).Seconds())
	}

	return Metrics{
		NodeID:            status.NodeID,
		Mode:              status.Mode,
		Online:            status.Enabled,
		UptimeSeconds:     uptime,
		Goroutines:        runtime.NumGoroutine(),
		MemAllocBytes:     mem.Alloc,
		MemSysBytes:       mem.Sys,
		HeapObjects:       mem.HeapObjects,
		GCCount:           mem.NumGC,
		NumCPU:            runtime.NumCPU(),
		PendingEmbeddings: status.PendingEmbeddings,
		VectorEntries:     entries,
		CacheItems:        status.CacheItems,
		CacheBackend:      "",
		EmbeddingModel:    status.EmbeddingModel,
		EmbeddingReady:    status.EmbeddingReady,
		RedisEnabled:      status.RedisEnabled,
		Backend:           status.Backend,
		DataDriver:        status.DataDriver,
		CollectedAt:       time.Now().UTC(),
	}
}
