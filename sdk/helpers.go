package sdk

import (
	"slices"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

// appendUnique adds items to an array only if they don't already exist in the array.
// Returns the modified array.
func appendUnique[I comparable](arr []I, item ...I) []I {
	for _, item := range item {
		if slices.Contains(arr, item) {
			return arr
		}
		arr = append(arr, item)
	}
	return arr
}

// doThisNotMoreThanOnceAnHour checks if an operation with the given key
// has been performed in the last hour. If not, it returns true and records
// the operation to prevent it from running again within the hour.
func doThisNotMoreThanOnceAnHour(key string) (doItNow bool) {
	_dtnmtoahLock.Lock()
	defer _dtnmtoahLock.Unlock()

	if _dtnmtoah == nil {
		// this runs only once for the lifetime of this library and
		// starts a long-running process of checking for expired items
		// and deleting them from this map every 10 minutes.
		_dtnmtoah = make(map[string]time.Time)
		go func() {
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

	_, hasBeenPerformedInTheLastHour := _dtnmtoah[key]
	if hasBeenPerformedInTheLastHour {
		return false
	}

	_dtnmtoah[key] = time.Now()
	return true
}
