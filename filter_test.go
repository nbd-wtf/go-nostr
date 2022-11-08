package nostr

import (
	"encoding/json"
	"testing"
	"time"

	"golang.org/x/exp/slices"
)

func TestFilterUnmarshal(t *testing.T) {
	raw := `{"ids": ["abc"],"#e":["zzz"],"#something":["nothing","bab"],"since":1644254609}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	if err != nil {
		t.Errorf("failed to parse filter json: %w", err)
	}

	if f.Since == nil || f.Since.Format("2006-01-02") != "2022-02-07" ||
		f.Until != nil ||
		f.Tags == nil || len(f.Tags) != 2 || !slices.Contains(f.Tags["something"], "bab") {
		t.Error("failed to parse filter correctly")
	}
}

func TestFilterMarshal(t *testing.T) {
	tm := time.Unix(12345678, 0)

	filterj, err := json.Marshal(Filter{
		Kinds: []int{1, 2, 4},
		Tags:  TagMap{"fruit": {"banana", "mango"}},
		Until: &tm,
	})
	if err != nil {
		t.Errorf("failed to marshal filter json: %w", err)
	}

	expected := `{"kinds":[1,2,4],"until":12345678,"#fruit":["banana","mango"]}`
	if string(filterj) != expected {
		t.Errorf("filter json was wrong: %s != %s", string(filterj), expected)
	}
}

func TestFilterMatching(t *testing.T) {
	if (Filter{Kinds: []int{4, 5}}).Matches(&Event{Kind: 6}) {
		t.Error("matched event that shouldn't have matched")
	}

	if !(Filter{Kinds: []int{4, 5}}).Matches(&Event{Kind: 4}) {
		t.Error("failed to match event by kind")
	}

	if !(Filter{
		Kinds: []int{4, 5},
		Tags: TagMap{
			"p": {"ooo"},
		},
		IDs: []string{"prefix"},
	}).Matches(&Event{
		Kind: 4,
		Tags: Tags{{"p", "ooo", ",x,x,"}, {"m", "yywyw", "xxx"}},
		ID:   "prefix123",
	}) {
		t.Error("failed to match event by kind+tags+id prefix")
	}
}

func TestFilterEquality(t *testing.T) {
	if !FilterEqual(
		Filter{Kinds: []int{4, 5}},
		Filter{Kinds: []int{4, 5}},
	) {
		t.Error("kinds filters should be equal")
	}

	if !FilterEqual(
		Filter{Kinds: []int{4, 5}, Tags: TagMap{"letter": {"a", "b"}}},
		Filter{Kinds: []int{4, 5}, Tags: TagMap{"letter": {"b", "a"}}},
	) {
		t.Error("kind+tags filters should be equal")
	}

	tm := time.Now()
	if !FilterEqual(
		Filter{
			Kinds: []int{4, 5},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
		Filter{
			Kinds: []int{5, 4},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
	) {
		t.Error("kind+2tags+since+ids filters should be equal")
	}

	if FilterEqual(
		Filter{Kinds: []int{1, 4, 5}},
		Filter{Kinds: []int{4, 5, 6}},
	) {
		t.Error("kinds filters shouldn't be equal")
	}
}
