package memory

import (
	"sync"

	"github.com/nbd-wtf/go-nostr/sdk/kvstore"
)

var _ kvstore.KVStore = (*Store)(nil)

type Store struct {
	sync.RWMutex
	data map[string][]byte
}

func NewStore() *Store {
	return &Store{
		data: make(map[string][]byte),
	}
}

func (s *Store) Get(key []byte) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	if val, ok := s.data[string(key)]; ok {
		// return a copy to prevent modification of stored data
		cp := make([]byte, len(val))
		copy(cp, val)
		return cp, nil
	}
	return nil, nil
}

func (s *Store) Set(key []byte, value []byte) error {
	s.Lock()
	defer s.Unlock()

	// store a copy to prevent modification of stored data
	cp := make([]byte, len(value))
	copy(cp, value)
	s.data[string(key)] = cp
	return nil
}

func (s *Store) Delete(key []byte) error {
	s.Lock()
	defer s.Unlock()
	delete(s.data, string(key))
	return nil
}

func (s *Store) Close() error {
	s.Lock()
	defer s.Unlock()
	s.data = nil
	return nil
}

func (s *Store) Update(key []byte, f func([]byte) ([]byte, error)) error {
	s.Lock()
	defer s.Unlock()

	var val []byte
	if v, ok := s.data[string(key)]; ok {
		// Return a copy to prevent modification of stored data
		val = make([]byte, len(v))
		copy(val, v)
	}

	newVal, err := f(val)
	if err == kvstore.NoOp {
		return nil
	} else if err != nil {
		return err
	}

	if newVal == nil {
		delete(s.data, string(key))
	} else {
		// Store a copy to prevent modification of stored data
		cp := make([]byte, len(newVal))
		copy(cp, newVal)
		s.data[string(key)] = cp
	}
	return nil
}
