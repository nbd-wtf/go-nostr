package nostr

import (
	"testing"
)

func TestTagHelpers(t *testing.T) {
	tags := Tags{
		Tag{"x"},
		Tag{"p", "abcdef", "wss://x.com"},
		Tag{"p", "123456", "wss://y.com"},
		Tag{"e", "eeeeee"},
		Tag{"e", "ffffff"},
	}

	if tags.GetFirst([]string{"x"}) == nil {
		t.Error("failed to get existing prefix")
	}
	if tags.GetFirst([]string{"x", ""}) != nil {
		t.Error("got with wrong prefix")
	}
	if tags.GetFirst([]string{"p", "abcdef", "wss://"}) == nil {
		t.Error("failed to get with existing prefix")
	}
	if tags.GetFirst([]string{"p", "abcdef", ""}) == nil {
		t.Error("failed to get with existing prefix (blank last string)")
	}
	if (*(tags.GetLast([]string{"e"})))[1] != "ffffff" {
		t.Error("failed to get last")
	}

	if len(tags.GetAll([]string{"e", ""})) != 2 {
		t.Error("failed to get all")
	}

	if len(tags.AppendUnique(Tag{"e", "ffffff"})) != 5 {
		t.Error("append unique changed the array size when existed")
	}
	if len(tags.AppendUnique(Tag{"e", "bbbbbb"})) != 6 {
		t.Error("append unique failed to append when didn't exist")
	}
	if tags.AppendUnique(Tag{"e", "eeeeee"})[4][1] != "ffffff" {
		t.Error("append unique changed the order")
	}
	if tags.AppendUnique(Tag{"e", "eeeeee"})[3][1] != "eeeeee" {
		t.Error("append unique changed the order")
	}
}
