package sdk

import (
	"sync"
	"time"
)

var (
	_dtnmtoah     map[string]time.Time
	_dtnmtoahLock sync.Mutex
)

func DoThisNotMoreThanOnceAnHour(key string) (doItNow bool) {
	if _dtnmtoah == nil {
		go func() {
			_dtnmtoah = make(map[string]time.Time)
			for {
				time.Sleep(time.Minute * 10)
				_dtnmtoahLock.Lock()
				now := time.Now()
				for k, v := range _dtnmtoah {
					if v.Before(now) {
						delete(_dtnmtoah, k)
					}
				}
				_dtnmtoahLock.Unlock()
			}
		}()
	}

	_dtnmtoahLock.Lock()
	defer _dtnmtoahLock.Unlock()

	_, exists := _dtnmtoah[key]
	return !exists
}

var serial = 0

func pickNext(list []string) string {
	serial++
	return list[serial%len(list)]
}
