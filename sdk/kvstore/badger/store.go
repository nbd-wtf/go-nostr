package badger

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/nbd-wtf/go-nostr/sdk/kvstore"
)

var _ kvstore.KVStore = (*Store)(nil)

type Store struct {
	db *badger.DB
}

func NewStore(path string) (*Store, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			valCopy = make([]byte, len(val))
			copy(valCopy, val)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return valCopy, nil
}

func (s *Store) Set(key []byte, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Update(key []byte, f func([]byte) ([]byte, error)) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var val []byte
		item, err := txn.Get(key)
		if err == nil {
			err = item.Value(func(v []byte) error {
				val = make([]byte, len(v))
				copy(val, v)
				return nil
			})
			if err != nil {
				return err
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		newVal, err := f(val)
		if err == kvstore.NoOp {
			return nil
		} else if err != nil {
			return err
		}

		if newVal == nil {
			return txn.Delete(key)
		}
		return txn.Set(key, newVal)
	})
}
