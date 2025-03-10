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
		return val, nil
	}
	return nil, nil
}

func (s *Store) Set(key []byte, value []byte) error {
	s.Lock()
	defer s.Unlock()

	s.data[string(key)] = value
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

	val, _ := s.data[string(key)]
	newVal, err := f(val)
	if err == kvstore.NoOp {
		return nil
	} else if err != nil {
		return err
	}

	if newVal == nil {
		delete(s.data, string(key))
	} else {
		s.data[string(key)] = newVal
	}
	return nil
}
