package kvstore

// KVStore is a simple key-value store interface
type KVStore interface {
	// Get retrieves a value for a given key. Returns nil if not found.
	Get(key []byte) ([]byte, error)

	// Set stores a value for a given key
	Set(key []byte, value []byte) error

	// Delete removes a key and its value
	Delete(key []byte) error

	// Close releases any resources held by the store
	Close() error

	// Update atomically modifies a value for a given key.
	// The function f receives the current value (nil if not found)
	// and returns the new value to be set.
	// If f returns nil, the key is deleted.
	Update(key []byte, f func([]byte) ([]byte, error)) error
}
