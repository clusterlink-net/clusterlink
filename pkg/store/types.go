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

package store

// Manager of multiple object stores that are persisted together.
type Manager interface {
	// GetObjectStore returns a store for a specific object type.
	GetObjectStore(name string, sampleObject any) ObjectStore
}

// ObjectStore represents a persistent store of objects.
type ObjectStore interface {
	// Create an object.
	// Returns ObjectExistsError if object already exists.
	Create(name string, object any) error
	// Update an object.
	// Returns ObjectNotFoundError if object does not exist.
	Update(name string, mutator func(any) any) error
	// Delete an object identified by the given name.
	Delete(name string) error
	// GetAll returns all of the objects in the store.
	GetAll() ([]any, error)
}

// ObjectExistsError represents an error caused due to an object which exists.
type ObjectExistsError struct{}

func (e *ObjectExistsError) Error() string {
	return "object already exists"
}

// ObjectNotFoundError represents an error caused due to an object which does not exist.
type ObjectNotFoundError struct{}

func (e *ObjectNotFoundError) Error() string {
	return "object not found"
}
