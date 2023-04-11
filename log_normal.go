//go:build !debug

package nostr

func debugLog(str string, args ...any) {
	return
}
