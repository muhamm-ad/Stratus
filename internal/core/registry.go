package core

import (
	"sort"
	"sync"
)

// Registry maps provider IDs to already-constructed connectors. It is safe for
// concurrent use (Wails calls in from multiple goroutines).
//
// It lives in core because it only ever references core types (CloudProvider,
// ProviderID); keeping it here means provider packages depend on a single
// package (core) and never on each other or on a "providers" parent package.
type Registry struct {
	mu sync.RWMutex
	m  map[CloudProviderID]CloudProvider
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{m: make(map[CloudProviderID]CloudProvider)}
}

// Register adds (or replaces) a connector under its own ID.
func (r *Registry) Register(c CloudProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[c.ID()] = c
}

// Get returns the connector for id, and whether it was found.
func (r *Registry) Get(id CloudProviderID) (CloudProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	return c, ok
}

// GetAll returns all the connectors in the registry.
func (r *Registry) GetAll() map[CloudProviderID]CloudProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.m
}

// IDs returns the registered provider IDs in a stable, sorted order.
func (r *Registry) IDs() []CloudProviderID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]CloudProviderID, 0, len(r.m))
	for id := range r.m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
