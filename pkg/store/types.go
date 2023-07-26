package store

// ObjectStore represents a persistent store of objects.
type ObjectStore interface {
	// Store an object.
	Store(name string, object any) error
	// Delete an object identified by the given name.
	Delete(name string) error
	// GetAll returns all of the objects in the store.
	GetAll() ([]any, error)
}
