package kv

// Store represents a key-value store.
type Store interface {
	// Put a (key, value) to the store.
	Put(key, value []byte) error
	// Delete a key (with its respective value) from the store.
	Delete(key []byte) error
	// Range calls f sequentially for each (key, value) where key starts with the given prefix.
	Range(prefix []byte, f func(key, value []byte) error) error
	// Close frees all resources (e.g. file handles, network sockets) used by the Store.
	Close() error
}
