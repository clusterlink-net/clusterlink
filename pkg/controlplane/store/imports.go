package store

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// Imports is a cached persistent store of imports.
type Imports struct {
	lock  sync.RWMutex
	cache map[string]*Import
	store store.ObjectStore

	logger *logrus.Entry
}

// Create an import.
func (s *Imports) Create(imp *Import) error {
	s.logger.Infof("Creating: '%s'.", imp.Name)

	if imp.Version > importStructVersion {
		return fmt.Errorf("incompatible import version %d, expected: %d",
			imp.Version, importStructVersion)
	}

	// persist to store
	if err := s.store.Create(imp.Name, imp); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[imp.Name] = imp
	return nil
}

// Update an import.
func (s *Imports) Update(name string, mutator func(*Import) *Import) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	var imp *Import
	err := s.store.Update(name, func(a any) any {
		imp = mutator(a.(*Import))
		return imp
	})
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[name] = imp
	return nil
}

// Get an import.
func (s *Imports) Get(name string) *Import {
	s.logger.Debugf("Getting '%s'.", name)

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.cache[name]
}

// Delete an import.
func (s *Imports) Delete(name string) (*Import, error) {
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

// GetAll returns all imports in the cache.
func (s *Imports) GetAll() []*Import {
	s.logger.Debug("Getting all imports.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	imports := make([]*Import, 0, len(s.cache))
	for _, imp := range s.cache {
		imports = append(imports, imp)
	}

	return imports
}

// Len returns the number of cached imports.
func (s *Imports) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.cache)
}

// init loads the cache with items from the backing store.
func (s *Imports) init() error {
	s.logger.Info("Initializing.")

	// get all imports from backing store
	imports, err := s.store.GetAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store all imports to the cache
	for _, object := range imports {
		imp := object.(*Import)
		s.cache[imp.Name] = imp
	}

	return nil
}

// NewImports returns a new cached store of imports.
func NewImports(manager store.Manager) (*Imports, error) {
	logger := logrus.WithField("component", "controlplane.store.imports")

	imports := &Imports{
		cache:  make(map[string]*Import),
		store:  manager.GetObjectStore(importStoreName, Import{}),
		logger: logger,
	}

	if err := imports.init(); err != nil {
		return nil, err
	}

	return imports, nil
}
