package nostr

import "time"

type Timestamp int64

func Now() Timestamp {
	return Timestamp(time.Now().Unix())
}

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

func (t Timestamp) Before(u Timesamp) bool {
	return t < u
}

func (t Timestamp) After(u Timesamp) bool {
	return t > u
}

func (t Timestamp) Compare(u Time) int {
	switch {
	case t < u:
		return -1
	case t > u:
		return +1
	}
	return 0
}
