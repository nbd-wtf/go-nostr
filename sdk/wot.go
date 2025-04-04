package sdk

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/FastFilter/xorfilter"
	"golang.org/x/sync/errgroup"
)

func PubKeyToShid(pubkey string) uint64 {
	shid, _ := strconv.ParseUint(pubkey[32:48], 16, 64)
	return shid
}

type wotCall struct {
	id          uint64 // basically the pubkey we're targeting here
	mutex       sync.Mutex
	resultbacks []chan WotXorFilter // all callers waiting for results
	errorbacks  []chan error        // all callers waiting for errors
	done        chan struct{}       // this is closed when this call is fully resolved and deleted
}

const wotCallsSize = 8

var (
	wotCallsMutex   sync.Mutex
	wotCallsInPlace [wotCallsSize]*wotCall
)

func (sys *System) LoadWoTFilter(ctx context.Context, pubkey string) (WotXorFilter, error) {
	id := PubKeyToShid(pubkey)
	pos := int(id % wotCallsSize)

start:
	wotCallsMutex.Lock()
	wc := wotCallsInPlace[pos]
	if wc == nil {
		// we are the first to call at this position
		wc = &wotCall{
			id:          id,
			resultbacks: make([]chan WotXorFilter, 0),
			errorbacks:  make([]chan error, 0),
			done:        make(chan struct{}),
		}
		wotCallsInPlace[pos] = wc
		wotCallsMutex.Unlock()
		goto actualcall
	} else {
		wotCallsMutex.Unlock()
	}

	wc.mutex.Lock()
	if wc.id == id {
		// there is already a call for this exact pubkey ongoing, so we just wait
		resch := make(chan WotXorFilter)
		errch := make(chan error)
		wc.resultbacks = append(wc.resultbacks, resch)
		wc.errorbacks = append(wc.errorbacks, errch)
		wc.mutex.Unlock()
		select {
		case res := <-resch:
			return res, nil
		case err := <-errch:
			return WotXorFilter{}, err
		}
	} else {
		wc.mutex.Unlock()
		// there is already a call in this place, but it's for a different pubkey, so wait
		<-wc.done
		// when it's done restart
		goto start
	}

actualcall:
	var res WotXorFilter
	m, err := sys.loadWoT(ctx, pubkey)
	if err != nil {
		wc.mutex.Lock()
		for _, ch := range wc.errorbacks {
			ch <- err
		}
	} else {
		res = makeWoTFilter(m)
		wc.mutex.Lock()
		for _, ch := range wc.resultbacks {
			ch <- res
		}
	}

	wotCallsMutex.Lock()
	wotCallsInPlace[pos] = nil
	wc.mutex.Unlock()
	close(wc.done)
	wotCallsMutex.Unlock()

	return res, err
}

func (sys *System) loadWoT(ctx context.Context, pubkey string) (chan string, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(45)

	res := make(chan string)

	// process follow lists
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for _, f := range sys.FetchFollowList(ctx, pubkey).Items {
			wg.Add(1)

			g.Go(func() error {
				res <- f.Pubkey

				ctx, cancel := context.WithTimeout(ctx, time.Second*7)
				defer cancel()

				ff := sys.FetchFollowList(ctx, f.Pubkey).Items
				for _, f2 := range ff {
					res <- f2.Pubkey
				}
				wg.Done()
				return nil
			})
		}

		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(res)
	}()

	return res, nil
}

func makeWoTFilter(m chan string) WotXorFilter {
	shids := make([]uint64, 0, 60000)
	shidMap := make(map[uint64]struct{}, 60000)
	for pk := range m {
		shid := PubKeyToShid(pk)
		if _, alreadyAdded := shidMap[shid]; !alreadyAdded {
			shidMap[shid] = struct{}{}
			shids = append(shids, shid)
		}
	}

	xf, _ := xorfilter.Populate(shids)
	return WotXorFilter{len(shids), *xf}
}

type WotXorFilter struct {
	Items int
	xorfilter.Xor8
}

func (wxf WotXorFilter) Contains(pubkey string) bool {
	if wxf.Items == 0 {
		return false
	}
	return wxf.Xor8.Contains(PubKeyToShid(pubkey))
}
