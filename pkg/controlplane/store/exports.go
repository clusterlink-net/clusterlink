package store

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-org/clusterlink/pkg/store"
)

// Exports is a cached persistent store of exports.
type Exports struct {
	lock  sync.RWMutex
	cache map[string]*Export
	store store.ObjectStore

	logger *logrus.Entry
}

// Create an export.
func (s *Exports) Create(export *Export) error {
	s.logger.Infof("Creating: '%s'.", export.Name)

	if export.Version > exportStructVersion {
		return fmt.Errorf("incompatible export version %d, expected: %d",
			export.Version, exportStructVersion)
	}

	// persist to store
	if err := s.store.Create(export.Name, export); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[export.Name] = export
	return nil
}

// Update an export.
func (s *Exports) Update(name string, mutator func(*Export) *Export) error {
	s.logger.Infof("Updating: '%s'.", name)

	// persist to store
	var export *Export
	err := s.store.Update(name, func(a any) any {
		export = mutator(a.(*Export))
		return export
	})
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store in cache
	s.cache[name] = export
	return nil
}

// Get an export.
func (s *Exports) Get(name string) *Export {
	s.logger.Debugf("Getting '%s'.", name)

	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.cache[name]
}

// Delete an export.
func (s *Exports) Delete(name string) (*Export, error) {
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

// GetAll returns all exports in the cache.
func (s *Exports) GetAll() []*Export {
	s.logger.Debug("Getting all exports.")

	s.lock.RLock()
	defer s.lock.RUnlock()

	exports := make([]*Export, 0, len(s.cache))
	for _, export := range s.cache {
		exports = append(exports, export)
	}

	return exports
}

// Len returns the number of cached exports.
func (s *Exports) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.cache)
}

// init loads the cache with items from the backing store.
func (s *Exports) init() error {
	s.logger.Info("Initializing.")

	// get all exports from backing store
	exports, err := s.store.GetAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// store all exports to the cache
	for _, object := range exports {
		export := object.(*Export)
		s.cache[export.Name] = export
	}

	return nil
}

// NewExports returns a new cached store of exports.
func NewExports(manager store.Manager) (*Exports, error) {
	logger := logrus.WithField("component", "controlplane.store.exports")

	exports := &Exports{
		cache:  make(map[string]*Export),
		store:  manager.GetObjectStore(exportStoreName, Export{}),
		logger: logger,
	}

	if err := exports.init(); err != nil {
		return nil, err
	}

	return exports, nil
}
