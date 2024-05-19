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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// ObjectStore represents a persistent store of objects, backed by a KV-store.
// Key format for an object is: <storeName>.<objectName>.
type ObjectStore struct {
	store Store

	keyPrefix  string
	objectType reflect.Type

	logger *logrus.Entry
}

// kvKey encodes object keys to a single key identifying the object in the store.
func (s *ObjectStore) kvKey(name string) []byte {
	return []byte(s.keyPrefix + name)
}

// Create an object.
func (s *ObjectStore) Create(name string, value any) error {
	s.logger.Infof("Creating: '%s'.", name)

	// serialize
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("unable to serialize object: %w", err)
	}

	// persist to store
	if err := s.store.Create(s.kvKey(name), encoded); err != nil {
		var keyExistsError *KeyExistsError
		if errors.As(err, &keyExistsError) {
			return &store.ObjectExistsError{}
		}
		return err
	}

	return nil
}

// Update an object.
func (s *ObjectStore) Update(name string, mutator func(any) any) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	err := s.store.Update(s.kvKey(name), func(value []byte) ([]byte, error) {
		// de-serialize old value
		decoded := reflect.New(s.objectType).Interface()
		if err := json.Unmarshal(value, decoded); err != nil {
			return nil, fmt.Errorf("unable to decode value for object '%s': %w", name, err)
		}

		// serialize mutated value
		encoded, err := json.Marshal(mutator(decoded))
		if err != nil {
			return nil, fmt.Errorf("unable to serialize mutated object '%s': %w", name, err)
		}

		return encoded, nil
	})
	if err != nil {
		var keyNotFoundError *KeyNotFoundError
		if errors.As(err, &keyNotFoundError) {
			return &store.ObjectNotFoundError{}
		}
		return err
	}

	return nil
}

// Delete an object identified by the given name.
func (s *ObjectStore) Delete(name string) error {
	s.logger.Infof("Deleting: '%s'.", name)
	return s.store.Delete(s.kvKey(name))
}

// GetAll returns all of the objects in the store.
func (s *ObjectStore) GetAll() ([]any, error) {
	s.logger.Info("Getting all objects.")

	var objects []any
	err := s.store.Range([]byte(s.keyPrefix), func(key, value []byte) error {
		s.logger.Debugf("De-serializing: %v.", value)

		decoded := reflect.New(s.objectType).Interface()
		if err := json.Unmarshal(value, decoded); err != nil {
			return fmt.Errorf("unable to decode object for key %v: %w", key, err)
		}

		objects = append(objects, decoded)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return objects, nil
}

// NewObjectStore returns a new object store backed by a KV-store.
func NewObjectStore(name string, s Store, sampleObject any) *ObjectStore {
	logger := logrus.WithFields(logrus.Fields{
		"component": "store.kv.object-store",
		"name":      name,
	})

	return &ObjectStore{
		store:      s,
		keyPrefix:  name + ".",
		objectType: reflect.TypeOf(sampleObject),
		logger:     logger,
	}
}
