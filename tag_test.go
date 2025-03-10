package nostr

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagHelpers(t *testing.T) {
	tags := Tags{
		Tag{"x"},
		Tag{"p", "abcdef", "wss://x.com"},
		Tag{"p", "123456", "wss://y.com"},
		Tag{"e", "eeeeee"},
		Tag{"e", "ffffff"},
	}

	assert.Nil(t, tags.Find("x"), "Find shouldn't have returned a tag with a single item")
	assert.NotNil(t, tags.FindWithValue("p", "abcdef"), "failed to get with existing prefix")
	assert.Equal(t, "ffffff", tags.FindLast("e")[1], "failed to get last")
	assert.Equal(t, 2, len(slices.Collect(tags.FindAll("e"))), "failed to get all")
	c := make(Tags, 0, 2)
	for _, tag := range tags.All([]string{"e", ""}) {
		c = append(c, tag)
	}
}
