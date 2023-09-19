package kv

import "github.com/clusterlink-org/clusterlink/pkg/store"

// Manager of multiple object stores that are persisted together.
type Manager struct {
	store Store
}

// GetObjectStore returns a store for a specific object type.
func (m *Manager) GetObjectStore(name string, sampleObject any) store.ObjectStore {
	return NewObjectStore(name, m.store, sampleObject)
}

// NewManager returns a new manager.
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}
