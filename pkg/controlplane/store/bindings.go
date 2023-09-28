package store

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// Bindings is a cached persistent store of bindings.
type Bindings struct {
	lock  sync.RWMutex
	cache map[string]map[string]*Binding
	store store.ObjectStore

	logger *logrus.Entry
}

// bindingName returns a unique name identifying the given binding.
func bindingName(binding *Binding) string {
	return fmt.Sprintf("%d.%s.%s", len(binding.Import), binding.Import, binding.Peer)
}

// Create a binding.
func (s *Bindings) Create(binding *Binding) error {
	s.logger.Infof("Creating: '%s'->'%s'.", binding.Import, binding.Peer)

	if binding.Version > bindingStructVersion {
		return fmt.Errorf("incompatible binding version %d, expected: %d",
			binding.Version, bindingStructVersion)
	}

	// persist to store
	if err := s.store.Create(bindingName(binding), binding); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	valMap, ok := s.cache[binding.Import]
	if !ok {
		valMap = make(map[string]*Binding)
		s.cache[binding.Import] = valMap
	}

	valMap[binding.Peer] = binding
	return nil
}

// Update a binding.
func (s *Bindings) Update(binding *Binding, mutator func(*Binding) *Binding) error {
	s.logger.Infof("Updating: '%s'->'%s'.", binding.Import, binding.Peer)

	// persist to store
	err := s.store.Update(bindingName(binding), func(a any) any {
		binding = mutator(a.(*Binding))
		return binding
	})
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	valMap, ok := s.cache[binding.Import]
	if !ok {
		valMap = make(map[string]*Binding)
		s.cache[binding.Import] = valMap
	}

	valMap[binding.Peer] = binding
	return nil
}

// Get all bindings for an import.
func (s *Bindings) Get(imp string) []*Binding {
	s.logger.Debugf("Getting all bindings for import '%s'.", imp)

	s.lock.RLock()
	defer s.lock.RUnlock()

	var bindings []*Binding
	if valMap, ok := s.cache[imp]; ok {
		bindings = make([]*Binding, 0)
		for _, val := range valMap {
			bindings = append(bindings, val)
		}
	}

	return bindings
}

// Delete a binding.
func (s *Bindings) Delete(binding *Binding) (*Binding, error) {
	s.logger.Infof("Deleting: '%s'->'%s'.", binding.Import, binding.Peer)

	// delete from store
	if err := s.store.Delete(bindingName(binding)); err != nil {
		return nil, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// delete from cache
	valMap, ok := s.cache[binding.Import]
	if !ok {
		return nil, nil
	}

	val, ok := valMap[binding.Peer]
	if !ok {
		return nil, nil
	}

	delete(valMap, binding.Peer)

	if len(valMap) == 0 {
		delete(s.cache, binding.Import)
	}

	return val, nil
}

// GetAll returns all bindings in the cache.
func (s *Bindings) GetAll() []*Binding {
	s.logger.Debug("Getting all bindings.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	var bindings []*Binding
	for _, m := range s.cache {
		for _, s := range m {
			bindings = append(bindings, s)
		}
	}

	return bindings
}

// Len returns the number of cached bindings.
func (s *Bindings) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	length := 0
	for _, m := range s.cache {
		length += len(m)
	}

	return length
}

// init loads the cache with items from the backing store.
func (s *Bindings) init() error {
	s.logger.Info("Initializing.")

	// get all bindings from backing store
	bindings, err := s.store.GetAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store all bindings to the cache
	for _, object := range bindings {
		binding := object.(*Binding)

		valMap, ok := s.cache[binding.Import]
		if !ok {
			valMap = make(map[string]*Binding)
			s.cache[binding.Import] = valMap
		}

		valMap[binding.Peer] = binding
	}

	return nil
}

// NewBindings returns a new cached store of bindings.
func NewBindings(manager store.Manager) (*Bindings, error) {
	logger := logrus.WithField("component", "controlplane.store.bindings")

	bindings := &Bindings{
		cache:  make(map[string]map[string]*Binding),
		store:  manager.GetObjectStore(bindingStoreName, Binding{}),
		logger: logger,
	}

	if err := bindings.init(); err != nil {
		return nil, err
	}

	return bindings, nil
}
