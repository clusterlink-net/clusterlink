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

package bolt

import (
	"bytes"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"

	"github.com/clusterlink-net/clusterlink/pkg/store/kv"
)

const (
	bucketName = "clink"
)

// Store implements a store backed by Bolt.
type Store struct {
	db *bbolt.DB

	logger *logrus.Entry
}

// Create a (key, value) in the store.
func (s *Store) Create(key, value []byte) error {
	s.logger.Debugf("Creating key: %v.", key)

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket.Get(key) != nil {
			return &kv.KeyExistsError{}
		}
		return bucket.Put(key, value)
	})
}

// Update a (key, value) in the store.
func (s *Store) Update(key []byte, mutator func([]byte) ([]byte, error)) error {
	s.logger.Debugf("Updating key: %v.", key)

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		value := bucket.Get(key)
		if value == nil {
			return &kv.KeyNotFoundError{}
		}

		// update value
		updated, err := mutator(value)
		if err != nil {
			return err
		}

		return bucket.Put(key, updated)
	})
}

// Delete a key (with its respective value) from the store.
func (s *Store) Delete(key []byte) error {
	s.logger.Debugf("Deleting key: %v.", key)

	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(bucketName)).Delete(key)
	})
}

// Range calls f sequentially for each (key, value) where key starts with the given prefix.
func (s *Store) Range(prefix []byte, f func(key, value []byte) error) error {
	s.logger.Infof("Iterating over all items with key prefix '%s'.", prefix)

	return s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte(bucketName)).Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			s.logger.Debugf("Read key from store: %v.", k)

			if err := f(k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// Close frees all resources (e.g. file handles, network sockets) used by the store.
func (s *Store) Close() error {
	s.logger.Info("Closing store.")
	return s.db.Close()
}

// Open a bolt store.
func Open(path string) (*Store, error) {
	// open
	db, err := bbolt.Open(path, 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to open store: %v", err)
	}

	// create the single bucket we use (if does not exist)
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create bucket: %v", err)
	}

	return &Store{
		db,
		logrus.WithField("component", "store.kv.bolt"),
	}, nil
}
