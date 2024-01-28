// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kv

import "github.com/clusterlink-net/clusterlink/pkg/store"

// Manager of multiple object stores that are persisted together.
type Manager struct {
	store Store
}

// GetObjectStore returns a store for a specific object type.
func (m *Manager) GetObjectStore(name string, sampleObject any) store.ObjectStore {
	return NewObjectStore(name, m.store, sampleObject)
}

// NewManager returns a new manager.
func NewManager(s Store) *Manager {
	return &Manager{store: s}
}
