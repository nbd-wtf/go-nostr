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
}
