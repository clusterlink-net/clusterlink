package store

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-org/clusterlink/pkg/store"
)

// AccessPolicies is a cached persistent store of Access Policies.
type AccessPolicies struct {
	lock  sync.RWMutex
	cache map[string]*AccessPolicy
	store store.ObjectStore

	logger *logrus.Entry
}

// Create an AccessPolicy.
func (s *AccessPolicies) Create(policy *AccessPolicy) error {
	s.logger.Infof("Creating: '%s'.", policy.Name)

	if policy.Version > accessPolicyStructVersion {
		return fmt.Errorf("incompatible access policy version %d, expected: %d",
			policy.Version, accessPolicyStructVersion)
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

// Update an access policy.
func (s *AccessPolicies) Update(name string, mutator func(*AccessPolicy) *AccessPolicy) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	var policy *AccessPolicy
	err := s.store.Update(name, func(a any) any {
		policy = mutator(a.(*AccessPolicy))
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

// Get an access policy.
func (s *AccessPolicies) Get(name string) *AccessPolicy {
	s.logger.Debugf("Getting '%s'.", name)

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.cache[name]
}

// Delete an access policy.
func (s *AccessPolicies) Delete(name string) (*AccessPolicy, error) {
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

// GetAll returns all access policies in the cache.
func (s *AccessPolicies) GetAll() []*AccessPolicy {
	s.logger.Debug("Getting all access policies.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	policies := make([]*AccessPolicy, 0, len(s.cache))
	for _, policy := range s.cache {
		policies = append(policies, policy)
	}

	return policies
}

// Len returns the number of cached access policies.
func (s *AccessPolicies) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.cache)
}

// init loads the cache with items from the backing store.
func (s *AccessPolicies) init() error {
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
		policy := object.(*AccessPolicy)
		s.cache[policy.Name] = policy
	}

	return nil
}

// NewAccessPolicies returns a new cached store of access policies.
func NewAccessPolicies(manager store.Manager) (*AccessPolicies, error) {
	logger := logrus.WithField("component", "controlplane.store.accesspolicies")

	policies := &AccessPolicies{
		cache:  make(map[string]*AccessPolicy),
		store:  manager.GetObjectStore(accessPolicyStoreName, AccessPolicy{}),
		logger: logger,
	}

	if err := policies.init(); err != nil {
		return nil, err
	}

	return policies, nil
}
