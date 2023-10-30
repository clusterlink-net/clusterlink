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

// LBPolicies is a cached persistent store of Load-Balancing Policies.
type LBPolicies struct {
	lock  sync.RWMutex
	cache map[string]*LBPolicy
	store store.ObjectStore

	logger *logrus.Entry
}

// Create a Load-Balancing Policy.
func (s *LBPolicies) Create(policy *LBPolicy) error {
	s.logger.Infof("Creating: '%s'.", policy.Name)

	if policy.Version > lbPolicyStructVersion {
		return fmt.Errorf("incompatible load-balancing policy version %d, expected: %d",
			policy.Version, lbPolicyStructVersion)
	}

	// persist to store
	if err := s.store.Create(policy.Name, policy); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[policy.Name] = policy
	return nil
}

// Update a load-balancing policy.
func (s *LBPolicies) Update(name string, mutator func(*LBPolicy) *LBPolicy) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	var policy *LBPolicy
	err := s.store.Update(name, func(a any) any {
		policy = mutator(a.(*LBPolicy))
		return policy
	})
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[name] = policy
	return nil
}

// Get a load-balancing policy.
func (s *LBPolicies) Get(name string) *LBPolicy {
	s.logger.Debugf("Getting '%s'.", name)

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.cache[name]
}

// Delete a load-balancing policy.
func (s *LBPolicies) Delete(name string) (*LBPolicy, error) {
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

// GetAll returns all load-balancing policies in the cache.
func (s *LBPolicies) GetAll() []*LBPolicy {
	s.logger.Debug("Getting all load-balancing policies.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	policies := make([]*LBPolicy, 0, len(s.cache))
	for _, policy := range s.cache {
		policies = append(policies, policy)
	}

	return policies
}

// Len returns the number of cached load-balancing policies.
func (s *LBPolicies) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.cache)
}

// init loads the cache with items from the backing store.
func (s *LBPolicies) init() error {
	s.logger.Info("Initializing.")

	// get all policies from backing store
	policies, err := s.store.GetAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store all policies to the cache
	for _, object := range policies {
		policy := object.(*LBPolicy)
		s.cache[policy.Name] = policy
	}

	return nil
}

// NewLBPolicies returns a new cached store of load-balancing policies.
func NewLBPolicies(manager store.Manager) (*LBPolicies, error) {
	logger := logrus.WithField("component", "controlplane.store.lbpolicies")

	policies := &LBPolicies{
		cache:  make(map[string]*LBPolicy),
		store:  manager.GetObjectStore(lbPolicyStoreName, LBPolicy{}),
		logger: logger,
	}

	if err := policies.init(); err != nil {
		return nil, err
	}

	return policies, nil
}
