package nip13

const (
	maxSafeAscii       = 126
	minSafeAscii       = 35
	availableSafeAscii = maxSafeAscii - minSafeAscii
)

func uintToStringCrazy(num uint64) string {
	nchars := 1 + num/availableSafeAscii
	chars := make([]byte, nchars)

	i := 0
	for {
		if num < availableSafeAscii {
			chars[i] = byte(num + minSafeAscii)
			break
		} else {
			chars[i] = byte(num/availableSafeAscii + minSafeAscii)
			num -= availableSafeAscii
			i++
		}
	}
	return string(chars)
}
