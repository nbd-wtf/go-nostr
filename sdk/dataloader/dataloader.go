package dataloader

import (
	"context"
	"errors"
	"sync"
	"time"
)

var NoValueError = errors.New("<dataloader: no value>")

// BatchFunc is a function, which when given a slice of keys (string), returns a map of `results` indexed by keys.
//
// The keys passed to this function are guaranteed to be unique.
type BatchFunc[K comparable, V any] func([]context.Context, []K) map[K]Result[V]

// Result is the data structure that a BatchFunc returns.
// It contains the resolved data, and any errors that may have occurred while fetching the data.
type Result[V any] struct {
	Data  V
	Error error
}

// ResultMany is used by the LoadMany method.
// It contains a list of resolved data and a list of errors.
// The lengths of the data list and error list will match, and elements at each index correspond to each other.
type ResultMany[V any] struct {
	Data  []V
	Error []error
}

// PanicErrorWrapper wraps the error interface.
// This is used to check if the error is a panic error.
// We should not cache panic errors.
type PanicErrorWrapper struct {
	panicError error
}

func (p *PanicErrorWrapper) Error() string {
	return p.panicError.Error()
}

// Loader implements the dataloader.Interface.
type Loader[K comparable, V any] struct {
	// the batch function to be used by this loader
	batchFn BatchFunc[K, V]

	// the maximum batch size. Set to 0 if you want it to be unbounded.
	batchCap int

	// count of queued up items
	count int

	// the maximum input queue size. Set to 0 if you want it to be unbounded.
	inputCap int

	// the amount of time to wait before triggering a batch
	wait time.Duration

	// lock to protect the batching operations
	batchLock sync.Mutex

	// current batcher
	curBatcher *batcher[K, V]

	// used to close the sleeper of the current batcher
	endSleeper chan bool

	// used by tests to prevent logs
	silent bool
}

// type used to on input channel
type batchRequest[K comparable, V any] struct {
	ctx     context.Context
	key     K
	channel chan Result[V]
}

// Option allows for configuration of Loader fields.
type Option[K comparable, V any] func(*Loader[K, V])

// WithBatchCapacity sets the batch capacity. Default is 0 (unbounded).
func WithBatchCapacity[K comparable, V any](c int) Option[K, V] {
	return func(l *Loader[K, V]) {
		l.batchCap = c
	}
}

// WithInputCapacity sets the input capacity. Default is 1000.
func WithInputCapacity[K comparable, V any](c int) Option[K, V] {
	return func(l *Loader[K, V]) {
		l.inputCap = c
	}
}

// WithWait sets the amount of time to wait before triggering a batch.
// Default duration is 16 milliseconds.
func WithWait[K comparable, V any](d time.Duration) Option[K, V] {
	return func(l *Loader[K, V]) {
		l.wait = d
	}
}

// withSilentLogger turns of log messages. It's used by the tests
func withSilentLogger[K comparable, V any]() Option[K, V] {
	return func(l *Loader[K, V]) {
		l.silent = true
	}
}

// NewBatchedLoader constructs a new Loader with given options.
func NewBatchedLoader[K comparable, V any](batchFn BatchFunc[K, V], opts ...Option[K, V]) *Loader[K, V] {
	loader := &Loader[K, V]{
		batchFn:  batchFn,
		inputCap: 1000,
		wait:     16 * time.Millisecond,
	}

	// Apply options
	for _, apply := range opts {
		apply(loader)
	}

	return loader
}

// Load load/resolves the given key, returning a channel that will contain the value and error.
// The first context passed to this function within a given batch window will be provided to
// the registered BatchFunc.
func (l *Loader[K, V]) Load(ctx context.Context, key K) (value V, err error) {
	c := make(chan Result[V], 1)

	// this is sent to batch fn. It contains the key and the channel to return
	// the result on
	req := &batchRequest[K, V]{ctx, key, c}

	l.batchLock.Lock()
	// start the batch window if it hasn't already started.
	if l.curBatcher == nil {
		l.curBatcher = l.newBatcher(l.silent)
		// start the current batcher batch function
		go l.curBatcher.batch()
		// start a sleeper for the current batcher
		l.endSleeper = make(chan bool)
		go l.sleeper(l.curBatcher, l.endSleeper)
	}

	l.curBatcher.input <- req

	// if we need to keep track of the count (max batch), then do so.
	if l.batchCap > 0 {
		l.count++
		// if we hit our limit, force the batch to start
		if l.count == l.batchCap {
			// end the batcher synchronously here because another call to Load
			// may concurrently happen and needs to go to a new batcher.
			l.curBatcher.end()
			// end the sleeper for the current batcher.
			// this is to stop the goroutine without waiting for the
			// sleeper timeout.
			close(l.endSleeper)
			l.reset()
		}
	}
	l.batchLock.Unlock()

	if v, ok := <-c; ok {
		return v.Data, v.Error
	}

	return value, NoValueError
}

func (l *Loader[K, V]) reset() {
	l.count = 0
	l.curBatcher = nil
}

type batcher[K comparable, V any] struct {
	input    chan *batchRequest[K, V]
	batchFn  BatchFunc[K, V]
	finished bool
	silent   bool
}

// newBatcher returns a batcher for the current requests
// all the batcher methods must be protected by a global batchLock
func (l *Loader[K, V]) newBatcher(silent bool) *batcher[K, V] {
	return &batcher[K, V]{
		input:   make(chan *batchRequest[K, V], l.inputCap),
		batchFn: l.batchFn,
		silent:  silent,
	}
}

// stop receiving input and process batch function
func (b *batcher[K, V]) end() {
	if !b.finished {
		close(b.input)
		b.finished = true
	}
}

// execute the batch of all items in queue
func (b *batcher[K, V]) batch() {
	var (
		ctxs = make([]context.Context, 0, 30)
		keys = make([]K, 0, 30)
		reqs = make([]*batchRequest[K, V], 0, 30)
		res  map[K]Result[V]
	)

	for item := range b.input {
		ctxs = append(ctxs, item.ctx)
		keys = append(keys, item.key)
		reqs = append(reqs, item)
	}

	func() {
		res = b.batchFn(ctxs, keys)
	}()

	for _, req := range reqs {
		if r, ok := res[req.key]; ok {
			req.channel <- r
		}
		close(req.channel)
	}
}

// wait the appropriate amount of time for the provided batcher
func (l *Loader[K, V]) sleeper(b *batcher[K, V], close chan bool) {
	select {
	// used by batch to close early. usually triggered by max batch size
	case <-close:
		return
	// this will move this goroutine to the back of the callstack?
	case <-time.After(l.wait):
	}

	// reset
	// this is protected by the batchLock to avoid closing the batcher input
	// channel while Load is inserting a request
	l.batchLock.Lock()
	b.end()

	// We can end here also if the batcher has already been closed and a
	// new one has been created. So reset the loader state only if the batcher
	// is the current one
	if l.curBatcher == b {
		l.reset()
	}
	l.batchLock.Unlock()
}
