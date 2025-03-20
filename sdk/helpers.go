package sdk

import (
	"slices"

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
