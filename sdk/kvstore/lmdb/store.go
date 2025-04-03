package lmdb

import (
	"os"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/nbd-wtf/go-nostr/sdk/kvstore"
)

var _ kvstore.KVStore = (*Store)(nil)

type Store struct {
	env *lmdb.Env
	dbi lmdb.DBI
}

func NewStore(path string) (*Store, error) {
	// create directory if it doesn't exist
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}

	// initialize environment
	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, err
	}

	// set max DBs and map size
	env.SetMaxDBs(1)
	env.SetMapSize(1 << 30) // 1GB

	// open the environment
	if err := env.Open(path, lmdb.NoTLS|lmdb.WriteMap, 0o644); err != nil {
		return nil, err
	}

	store := &Store{env: env}

	// open the database
	if err := env.Update(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI("store", lmdb.Create)
		if err != nil {
			return err
		}
		store.dbi = dbi
		return nil
	}); err != nil {
		env.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	var value []byte
	err := s.env.View(func(txn *lmdb.Txn) error {
		v, err := txn.Get(s.dbi, key)
		if lmdb.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		// make a copy since v is only valid during the transaction
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Store) Set(key []byte, value []byte) error {
	return s.env.Update(func(txn *lmdb.Txn) error {
		return txn.Put(s.dbi, key, value, 0)
	})
}

func (s *Store) Delete(key []byte) error {
	return s.env.Update(func(txn *lmdb.Txn) error {
		return txn.Del(s.dbi, key, nil)
	})
}

func (s *Store) Close() error {
	s.env.Close()
	return nil
}

func (s *Store) Update(key []byte, f func([]byte) ([]byte, error)) error {
	return s.env.Update(func(txn *lmdb.Txn) error {
		var val []byte
		v, err := txn.Get(s.dbi, key)
		if err == nil {
			// make a copy since v is only valid during the transaction
			val = make([]byte, len(v))
			copy(val, v)
		} else if !lmdb.IsNotFound(err) {
			return err
		}

		newVal, err := f(val)
		if err == kvstore.NoOp {
			return nil
		} else if err != nil {
			return err
		}

		if newVal == nil {
			return txn.Del(s.dbi, key, nil)
		}
		return txn.Put(s.dbi, key, newVal, 0)
	})
}
