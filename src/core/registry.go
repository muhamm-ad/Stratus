package core

import (
	"sort"
	"sync"
)

// Registry maps provider IDs to already-constructed connectors. It is safe for
// concurrent use (Wails calls in from multiple goroutines).
//
// It lives in core because it only ever references core types (ProviderConnector,
// ProviderID); keeping it here means provider packages depend on a single
// package (core) and never on each other or on a "providers" parent package.
type Registry struct {
	mu sync.RWMutex
	m  map[ProviderID]ProviderConnector
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{m: make(map[ProviderID]ProviderConnector)}
}

// Register adds (or replaces) a connector under its own ID.
func (r *Registry) Register(c ProviderConnector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[c.ID()] = c
}

// Get returns the connector for id, and whether it was found.
func (r *Registry) Get(id ProviderID) (ProviderConnector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	return c, ok
}

// IDs returns the registered provider IDs in a stable, sorted order.
func (r *Registry) IDs() []ProviderID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]ProviderID, 0, len(r.m))
	for id := range r.m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}