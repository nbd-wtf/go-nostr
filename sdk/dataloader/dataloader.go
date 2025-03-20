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

// Loader implements the dataloader.Interface.
type Loader[K comparable, V any] struct {
	// the batch function to be used by this loader
	batchFn BatchFunc[K, V]

	// the maximum batch size. Set to 0 if you want it to be unbounded.
	batchCap uint

	// count of queued up items
	count uint

	// the amount of time to wait before triggering a batch
	wait time.Duration

	// lock to protect the batching operations
	batchLock sync.Mutex

	// current batcher
	curBatcher *batcher[K, V]

	// used to close the sleeper of the current batcher
	thresholdReached chan bool
}

// type used to on input channel
type batchRequest[K comparable, V any] struct {
	ctx     context.Context
	key     K
	channel chan Result[V]
}

type Options struct {
	Wait         time.Duration
	MaxThreshold uint
}

// NewBatchedLoader constructs a new Loader with given options.
func NewBatchedLoader[K comparable, V any](batchFn BatchFunc[K, V], opts Options) *Loader[K, V] {
	loader := &Loader[K, V]{
		batchFn:  batchFn,
		batchCap: opts.MaxThreshold,
		count:    0,
		wait:     opts.Wait,
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
	req := batchRequest[K, V]{ctx, key, c}

	l.batchLock.Lock()
	// start the batch window if it hasn't already started.
	if l.curBatcher == nil {
		l.curBatcher = l.newBatcher()

		// start a sleeper for the current batcher
		l.thresholdReached = make(chan bool)

		// we will run the batch function either after some time or after a threshold has been reached
		b := l.curBatcher
		go func() {
			select {
			case <-l.thresholdReached:
			case <-time.After(l.wait):
			}

			// We can end here also if the batcher has already been closed and a
			// new one has been created. So reset the loader state only if the batcher
			// is the current one
			if l.curBatcher == b {
				l.reset()
			}

			var (
				ctxs = make([]context.Context, 0, len(b.requests))
				keys = make([]K, 0, len(b.requests))
				res  map[K]Result[V]
			)

			for _, item := range b.requests {
				ctxs = append(ctxs, item.ctx)
				keys = append(keys, item.key)
			}

			res = l.batchFn(ctxs, keys)

			for _, req := range b.requests {
				if r, ok := res[req.key]; ok {
					req.channel <- r
				}
				close(req.channel)
			}
		}()
	}

	l.curBatcher.requests = append(l.curBatcher.requests, req)

	l.count++
	if l.count == l.batchCap {
		close(l.thresholdReached)

		// end the batcher synchronously here because another call to Load
		// may concurrently happen and needs to go to a new batcher.
		l.reset()
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
	requests []batchRequest[K, V]
	batchFn  BatchFunc[K, V]
}

// newBatcher returns a batcher for the current requests
func (l *Loader[K, V]) newBatcher() *batcher[K, V] {
	return &batcher[K, V]{
		requests: make([]batchRequest[K, V], 0, l.batchCap),
		batchFn:  l.batchFn,
	}
}
