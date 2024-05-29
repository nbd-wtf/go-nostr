package relays

import (
	"sync"
	"unsafe"
)

const MAX_LOCKS = 50

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

func namedLock(name string) (unlock func()) {
	sptr := unsafe.StringData(name)
	idx := uint64(memhash(unsafe.Pointer(sptr), 0, uintptr(len(name)))) % MAX_LOCKS
	l := &namedMutexPool[idx]
	l.Lock()
	return l.Unlock
}
