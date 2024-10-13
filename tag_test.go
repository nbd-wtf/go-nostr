package nostr

import (
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

	assert.NotNil(t, tags.GetFirst([]string{"x"}), "failed to get existing prefix")
	assert.Nil(t, tags.GetFirst([]string{"x", ""}), "got with wrong prefix")
	assert.NotNil(t, tags.GetFirst([]string{"p", "abcdef", "wss://"}), "failed to get with existing prefix")
	assert.NotNil(t, tags.GetFirst([]string{"p", "abcdef", ""}), "failed to get with existing prefix (blank last string)")
	assert.Equal(t, "ffffff", (*(tags.GetLast([]string{"e"})))[1], "failed to get last")
	assert.Equal(t, 2, len(tags.GetAll([]string{"e", ""})), "failed to get all")
	c := make(Tags, 0, 2)
	for _, tag := range tags.All([]string{"e", ""}) {
		c = append(c, tag)
	}
	assert.Equal(t, tags.GetAll([]string{"e", ""}), c)
	assert.Equal(t, 5, len(tags.AppendUnique(Tag{"e", "ffffff"})), "append unique changed the array size when existed")
	assert.Equal(t, 6, len(tags.AppendUnique(Tag{"e", "bbbbbb"})), "append unique failed to append when didn't exist")
	assert.Equal(t, "ffffff", tags.AppendUnique(Tag{"e", "eeeeee"})[4][1], "append unique changed the order")
	assert.Equal(t, "eeeeee", tags.AppendUnique(Tag{"e", "eeeeee"})[3][1], "append unique changed the order")

	filtered := tags.FilterOut([]string{"e"})
	tags.FilterOutInPlace([]string{"e"})
	assert.ElementsMatch(t, filtered, tags)
	assert.Len(t, filtered, 3)
}
