package bolt

import (
	"bytes"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

const (
	bucketName = "clink"
)

// Store implements a store backed by Bolt.
type Store struct {
	db *bbolt.DB

	logger *logrus.Entry
}

// Put an item to store.
func (s *Store) Put(key, value []byte) error {
	s.logger.Debugf("Putting key: %v.", key)

	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(bucketName)).Put(key, value)
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
