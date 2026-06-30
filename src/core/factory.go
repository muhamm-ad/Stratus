package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Factory builds a connector from its raw config section (the provider's slice
// of config.json). Provider packages register one in init(); adding a new cloud
// provider is therefore a new package that calls RegisterProvider plus a blank
// import — no change to core or to the app wiring.
type Factory func(raw json.RawMessage) (ProviderConnector, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[ProviderID]Factory{}
)

// RegisterProvider adds a provider factory to the global catalog. Call it from
// a provider package's init(). Registering the same ID twice is a programming
// error and panics.
func RegisterProvider(id ProviderID, f Factory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	if _, dup := factories[id]; dup {
		panic(fmt.Sprintf("core: duplicate provider registration: %q", id))
	}
	factories[id] = f
}

// Factories returns a copy of the registered factory catalog (safe to range).
func Factories() map[ProviderID]Factory {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	out := make(map[ProviderID]Factory, len(factories))
	for id, f := range factories {
		out[id] = f
	}
	return out
}

// BuildAll constructs every provided factory from its matching config section
// and returns them in a Registry. A provider whose section is missing or
// invalid is reported in errs and skipped, so the app still runs with the
// others. Pass core.Factories() in production; pass a controlled map in tests.
func BuildAll(factories map[ProviderID]Factory, sections map[string]json.RawMessage) (*Registry, []error) {
	ids := make([]ProviderID, 0, len(factories))
	for id := range factories {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	reg := NewRegistry()
	var errs []error
	for _, id := range ids {
		c, err := factories[id](sections[string(id)])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
			continue
		}
		reg.Register(c)
	}
	return reg, errs
}
