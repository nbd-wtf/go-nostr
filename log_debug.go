//go:build debug

package nostr

import (
	"encoding/json"
	"fmt"
)

func debugLog(str string, args ...any) {
	for i, v := range args {
		switch v.(type) {
		case []json.RawMessage:
			j, _ := json.Marshal(v)
			args[i] = string(j)
		case []byte:
			args[i] = string(v.([]byte))
		case fmt.Stringer:
			args[i] = v.(fmt.Stringer).String()
		}
	}

	DebugLogger.Printf(str, args...)
}
