package store

// Manager of multiple object stores that are persisted together.
type Manager interface {
	// GetObjectStore returns a store for a specific object type.
	GetObjectStore(name string, sampleObject any) ObjectStore
}

// ObjectStore represents a persistent store of objects.
type ObjectStore interface {
	// Store an object.
	Store(name string, object any) error
	// Delete an object identified by the given name.
	Delete(name string) error
	// GetAll returns all of the objects in the store.
	GetAll() ([]any, error)
}
