package nip77

import (
	"sync"
)

type idlistpool struct {
	initialsize int
	pool        [][]string
	sync.Mutex
}

func newidlistpool(initialsize int) *idlistpool {
	ilp := idlistpool{
		initialsize: initialsize,
		pool:        make([][]string, 1, 2),
	}

	ilp.pool[0] = make([]string, 0, initialsize)

	return &ilp
}

func (ilp *idlistpool) grab() []string {
	ilp.Lock()
	defer ilp.Unlock()

	l := len(ilp.pool)
	if l > 0 {
		idlist := ilp.pool[l-1]
		ilp.pool = ilp.pool[0 : l-1]
		return idlist
	}
	idlist := make([]string, 0, ilp.initialsize)
	return idlist
}

func (ilp *idlistpool) giveback(idlist []string) {
	idlist = idlist[:0]
	ilp.pool = append(ilp.pool, idlist)
}
