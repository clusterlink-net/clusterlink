package kv

// Store represents a key-value store.
type Store interface {
	// Create a (key, value) in the store.
	// Returns KeyExistsError if key already exists.
	Create(key, value []byte) error
	// Update a (key, value) in the store.
	// Returns KeyNotFoundError if key does not exist.
	Update(key []byte, mutator func([]byte) ([]byte, error)) error
	// Delete a key (with its respective value) from the store.
	Delete(key []byte) error
	// Range calls f sequentially for each (key, value) where key starts with the given prefix.
	Range(prefix []byte, f func(key, value []byte) error) error
	// Close frees all resources (e.g. file handles, network sockets) used by the Store.
	Close() error
}

// KeyExistsError represents an error caused due to a key which exists.
type KeyExistsError struct{}

func (e *KeyExistsError) Error() string {
	return "key already exists"
}

// KeyNotFoundError represents an error caused due to a key which does not exist.
type KeyNotFoundError struct{}

func (e *KeyNotFoundError) Error() string {
	return "key not found"
}
