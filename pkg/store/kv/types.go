// Copyright (c) The ClusterLink Authors.
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

// Store represents a key-value store.
type Store interface {
	// Create a (key, value) in the store.
	// Returns KeyExistsError if key already exists.
	Create(key, value []byte) error
	// Update a (key, value) in the store.
	// Returns KeyNotFoundError if key does not exist.
	Update(key []byte, mutator func([]byte) ([]byte, error)) error
	// Delete a key (with its respective value) from the store.
	Delete(key []byte) error
	// Range calls f sequentially for each (key, value) where key starts with the given prefix.
	Range(prefix []byte, f func(key, value []byte) error) error
	// Close frees all resources (e.g. file handles, network sockets) used by the Store.
	Close() error
}

// KeyExistsError represents an error caused due to a key which exists.
type KeyExistsError struct{}

func (e *KeyExistsError) Error() string {
	return "key already exists"
}

// KeyNotFoundError represents an error caused due to a key which does not exist.
type KeyNotFoundError struct{}

func (e *KeyNotFoundError) Error() string {
	return "key not found"
}
