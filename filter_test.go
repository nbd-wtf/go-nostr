package nostr

import (
	"encoding/json"
	"testing"

	"golang.org/x/exp/slices"
)

func TestFilterUnmarshal(t *testing.T) {
	raw := `{"ids": ["abc"],"#e":["zzz"],"#something":["nothing","bab"],"since":1644254609,"search":"test"}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	if err != nil {
		t.Errorf("failed to parse filter json: %v", err)
	}

	if f.Since == nil || f.Since.Time().UTC().Format("2006-01-02") != "2022-02-07" ||
		f.Until != nil ||
		f.Tags == nil || len(f.Tags) != 2 || !slices.Contains(f.Tags["something"], "bab") ||
		f.Search != "test" {
		t.Error("failed to parse filter correctly")
	}
}

func TestFilterMarshal(t *testing.T) {
	until := Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds: []int{1, 2, 4},
		Tags:  TagMap{"fruit": {"banana", "mango"}},
		Until: &until,
	})
	if err != nil {
		t.Errorf("failed to marshal filter json: %v", err)
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

func TestFilterMatchingLive(t *testing.T) {
	var filter Filter
	var event Event

	json.Unmarshal([]byte(`{"kinds":[1],"authors":["a8171781fd9e90ede3ea44ddca5d3abf828fe8eedeb0f3abb0dd3e563562e1fc","1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","ed4ca520e9929dfe9efdadf4011b53d30afd0678a09aa026927e60e7a45d9244"],"since":1677033299}`), &filter)
	json.Unmarshal([]byte(`{"id":"5a127c9c931f392f6afc7fdb74e8be01c34035314735a6b97d2cf360d13cfb94","pubkey":"1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","created_at":1677033299,"kind":1,"tags":[["t","japan"]],"content":"If you like my art,I'd appreciate a coin or two!!\nZap is welcome!! Thanks.\n\n\n#japan #bitcoin #art #bananaart\nhttps://void.cat/d/CgM1bzDgHUCtiNNwfX9ajY.webp","sig":"828497508487ca1e374f6b4f2bba7487bc09fccd5cc0d1baa82846a944f8c5766918abf5878a580f1e6615de91f5b57a32e34c42ee2747c983aaf47dbf2a0255"}`), &event)

	if !filter.Matches(&event) {
		t.Error("live filter should match")
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

	tm := Now()
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
