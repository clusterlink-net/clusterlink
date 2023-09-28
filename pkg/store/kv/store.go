package kv

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// ObjectStore represents a persistent store of objects, backed by a KV-store.
// Key format for an object is: <storeName>.<objectName>
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
		return fmt.Errorf("unable to serialize object: %v", err)
	}

	// persist to store
	if err := s.store.Create(s.kvKey(name), encoded); err != nil {
		if _, ok := err.(*KeyExistsError); ok {
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
			return nil, fmt.Errorf("unable to decode value for object '%s': %v", name, err)
		}

		// serialize mutated value
		encoded, err := json.Marshal(mutator(decoded))
		if err != nil {
			return nil, fmt.Errorf("unable to serialize mutated object '%s': %v", name, err)
		}

		return encoded, nil
	})
	if err != nil {
		if _, ok := err.(*KeyNotFoundError); ok {
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
			return fmt.Errorf("unable to decode object for key %v: %v", key, err)
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
func NewObjectStore(name string, store Store, sampleObject any) *ObjectStore {
	logger := logrus.WithFields(logrus.Fields{
		"component": "store.kv.object-store",
		"name":      name,
	})

	return &ObjectStore{
		store:      store,
		keyPrefix:  name + ".",
		objectType: reflect.TypeOf(sampleObject),
		logger:     logger,
	}
}
