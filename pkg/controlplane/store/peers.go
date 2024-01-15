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

package store

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// Peers is a cached persistent store of peers.
type Peers struct {
	lock  sync.RWMutex
	cache map[string]*Peer
	store store.ObjectStore

	logger *logrus.Entry
}

// Create a peer.
func (s *Peers) Create(peer *Peer) error {
	s.logger.Infof("Creating: '%s'.", peer.Name)

	if peer.Version > peerStructVersion {
		return fmt.Errorf("incompatible peer version %d, expected: %d",
			peer.Version, peerStructVersion)
	}

	// persist to store
	if err := s.store.Create(peer.Name, peer); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[peer.Name] = peer
	return nil
}

// Update a peer.
func (s *Peers) Update(name string, mutator func(*Peer) *Peer) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	var peer *Peer
	err := s.store.Update(name, func(a any) any {
		peer = mutator(a.(*Peer))
		return peer
	})
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[name] = peer
	return nil
}

// Get a peer.
func (s *Peers) Get(name string) *Peer {
	s.logger.Debugf("Getting '%s'.", name)

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.cache[name]
}

// Delete a peer.
func (s *Peers) Delete(name string) (*Peer, error) {
	s.logger.Infof("Deleting: '%s'.", name)

	// delete from store
	if err := s.store.Delete(name); err != nil {
		return nil, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// delete from cache
	val := s.cache[name]
	delete(s.cache, name)
	return val, nil
}

// GetAll returns all peers in the cache.
func (s *Peers) GetAll() []*Peer {
	s.logger.Debug("Getting all peers.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	peers := make([]*Peer, 0, len(s.cache))
	for _, peer := range s.cache {
		peers = append(peers, peer)
	}

	return peers
}

// Len returns the number of cached peers.
func (s *Peers) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.cache)
}

// init loads the cache with items from the backing store.
func (s *Peers) init() error {
	s.logger.Info("Initializing.")

	// get all peers from backing store
	peers, err := s.store.GetAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store all peers to the cache
	for _, object := range peers {
		if peer, ok := object.(*Peer); ok {
			s.cache[peer.Name] = peer
		}
	}

	return nil
}

// NewPeers returns a new cached store of peers.
func NewPeers(manager store.Manager) (*Peers, error) {
	logger := logrus.WithField("component", "controlplane.store.peers")

	peers := &Peers{
		cache:  make(map[string]*Peer),
		store:  manager.GetObjectStore(peerStoreName, Peer{}),
		logger: logger,
	}

	if err := peers.init(); err != nil {
		return nil, err
	}

	return peers, nil
}
