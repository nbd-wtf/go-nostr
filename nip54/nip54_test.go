package nip54

import (
	"fmt"
	"testing"
)

func TestNormalization(t *testing.T) {
	for _, vector := range []struct {
		before string
		after  string
	}{
		{" hello  ", "hello"},
		{"Goodbye", "goodbye"},
		{"the long and winding road / that leads to your door", "the-long-and-winding-road---that-leads-to-your-door"},
		{"it's 平仮名", "it-s-平仮名"},
	} {
		if norm := NormalizeIdentifier(vector.before); norm != vector.after {
			fmt.Println([]byte(vector.after), []byte(norm))
			t.Fatalf("%s: %s != %s", vector.before, norm, vector.after)
		}
	}
}
