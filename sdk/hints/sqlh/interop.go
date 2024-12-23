package sqlh

import (
	"strconv"
	"strings"
)

type interop struct {
	maxFunc              string
	getUnixEpochFunc     string
	generateBindingSpots func(start, n int) string
}

var sqliteInterop = interop{
	maxFunc:          "max",
	getUnixEpochFunc: "unixepoch()",
	generateBindingSpots: func(_, n int) string {
		b := strings.Builder{}
		b.Grow(n * 2)
		for i := range n {
			if i == n-1 {
				b.WriteString("?")
			} else {
				b.WriteString("?,")
			}
		}
		return b.String()
	},
}

var postgresInterop = interop{
	maxFunc:          "greatest",
	getUnixEpochFunc: "extract(epoch from now())",
	generateBindingSpots: func(start, n int) string {
		b := strings.Builder{}
		b.Grow(n * 2)
		end := start + n
		for i := start; i < end; i++ {
			v := i + 1
			b.WriteRune('$')
			b.WriteString(strconv.Itoa(v))
			if i != end-1 {
				b.WriteRune(',')
			}
		}
		return b.String()
	},
}
