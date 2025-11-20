package nostr

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterUnmarshal(t *testing.T) {
	raw := `{"ids": ["abc"],"#e":["zzz"],"#something":["nothing","bab"],"since":1644254609,"search":"test"}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	assert.NoError(t, err)

	assert.Condition(t, func() (success bool) {
		if f.Since == nil || f.Since.Time().UTC().Format("2006-01-02") != "2022-02-07" ||
			f.Until != nil ||
			f.Tags == nil || len(f.Tags) != 2 || !slices.Contains(f.Tags["something"], "bab") ||
			f.Search != "test" {
			return false
		}
		return true
	}, "failed to parse filter correctly")
}

func TestFilterMarshal(t *testing.T) {
	until := Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds: []int{KindTextNote, KindRecommendServer, KindEncryptedDirectMessage},
		Tags:  TagMap{"fruit": {"banana", "mango"}},
		Until: &until,
	})
	assert.NoError(t, err)

	expected := `{"kinds":[1,2,4],"until":12345678,"#fruit":["banana","mango"]}`
	assert.Equal(t, expected, string(filterj))
}

func TestFilterUnmarshalWithLimitZero(t *testing.T) {
	raw := `{"ids": ["abc"],"#e":["zzz"],"limit":0,"#something":["nothing","bab"],"since":1644254609,"search":"test"}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	assert.NoError(t, err)

	assert.Condition(t, func() (success bool) {
		if f.Since == nil ||
			f.Since.Time().UTC().Format("2006-01-02") != "2022-02-07" ||
			f.Until != nil ||
			f.Tags == nil || len(f.Tags) != 2 || !slices.Contains(f.Tags["something"], "bab") ||
			f.Search != "test" ||
			f.LimitZero == false {
			return false
		}
		return true
	}, "failed to parse filter correctly")
}

func TestFilterMarshalWithLimitZero(t *testing.T) {
	until := Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds:     []int{KindTextNote, KindRecommendServer, KindEncryptedDirectMessage},
		Tags:      TagMap{"fruit": {"banana", "mango"}},
		Until:     &until,
		LimitZero: true,
	})
	assert.NoError(t, err)

	expected := `{"kinds":[1,2,4],"until":12345678,"limit":0,"#fruit":["banana","mango"]}`
	assert.Equal(t, expected, string(filterj))
}

func TestFilterMatchingLive(t *testing.T) {
	var filter Filter
	var event Event

	json.Unmarshal([]byte(`{"kinds":[1],"authors":["a8171781fd9e90ede3ea44ddca5d3abf828fe8eedeb0f3abb0dd3e563562e1fc","1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","ed4ca520e9929dfe9efdadf4011b53d30afd0678a09aa026927e60e7a45d9244"],"since":1677033299}`), &filter)
	json.Unmarshal([]byte(`{"id":"5a127c9c931f392f6afc7fdb74e8be01c34035314735a6b97d2cf360d13cfb94","pubkey":"1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","created_at":1677033299,"kind":1,"tags":[["t","japan"]],"content":"If you like my art,I'd appreciate a coin or two!!\nZap is welcome!! Thanks.\n\n\n#japan #bitcoin #art #bananaart\nhttps://void.cat/d/CgM1bzDgHUCtiNNwfX9ajY.webp","sig":"828497508487ca1e374f6b4f2bba7487bc09fccd5cc0d1baa82846a944f8c5766918abf5878a580f1e6615de91f5b57a32e34c42ee2747c983aaf47dbf2a0255"}`), &event)

	assert.True(t, filter.Matches(&event), "live filter should match")
}

func TestFilterEquality(t *testing.T) {
	assert.True(t, FilterEqual(
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}},
	), "kinds filters should be equal")

	assert.True(t, FilterEqual(
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}, Tags: TagMap{"letter": {"a", "b"}}},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}, Tags: TagMap{"letter": {"b", "a"}}},
	), "kind+tags filters should be equal")

	tm := Now()
	assert.True(t, FilterEqual(
		Filter{
			Kinds: []int{KindEncryptedDirectMessage, KindDeletion},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
		Filter{
			Kinds: []int{KindDeletion, KindEncryptedDirectMessage},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
	), "kind+2tags+since+ids filters should be equal")

	assert.False(t, FilterEqual(
		Filter{Kinds: []int{KindTextNote, KindEncryptedDirectMessage, KindDeletion}},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion, KindRepost}},
	), "kinds filters shouldn't be equal")
}

func TestFilterClone(t *testing.T) {
	ts := Now() - 60*60
	flt := Filter{
		Kinds: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
		Since: &ts,
		IDs:   []string{"9894b4b5cb5166d23ee8899a4151cf0c66aec00bde101982a13b8e8ceb972df9"},
	}
	clone := flt.Clone()
	assert.True(t, FilterEqual(flt, clone), "clone is not equal:\n %v !=\n %v", flt, clone)

	clone1 := flt.Clone()
	clone1.IDs = append(clone1.IDs, "88f0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	assert.False(t, FilterEqual(flt, clone1), "modifying the clone ids should cause it to not be equal anymore")

	clone2 := flt.Clone()
	clone2.Tags["letter"] = append(clone2.Tags["letter"], "c")
	assert.False(t, FilterEqual(flt, clone2), "modifying the clone tag items should cause it to not be equal anymore")

	clone3 := flt.Clone()
	clone3.Tags["g"] = []string{"drt"}
	assert.False(t, FilterEqual(flt, clone3), "modifying the clone tag map should cause it to not be equal anymore")

	clone4 := flt.Clone()
	*clone4.Since++
	assert.False(t, FilterEqual(flt, clone4), "modifying the clone since should cause it to not be equal anymore")
}

func TestTheoreticalLimit(t *testing.T) {
	require.Equal(t, 6, GetTheoreticalLimit(Filter{IDs: []string{"a", "b", "c", "d", "e", "f"}}))
	require.Equal(t, 9, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c"}, Kinds: []int{3, 0, 10002}}))
	require.Equal(t, 4, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d"}, Kinds: []int{10050}}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d"}}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Kinds: []int{3, 0, 10002}}))
	require.Equal(t, 24, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d", "e", "f"}, Kinds: []int{30023, 30024}, Tags: TagMap{"d": []string{"aaa", "bbb"}}}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d", "e", "f"}, Kinds: []int{30023, 30024}}))
}

func TestFilterUnmarshalWithAndTags(t *testing.T) {
	raw := `{"kinds":[1],"&t":["meme","cat"],"#t":["black","white"]}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	assert.NoError(t, err)

	assert.Condition(t, func() (success bool) {
		if f.Kinds == nil || len(f.Kinds) != 1 || f.Kinds[0] != 1 {
			return false
		}
		if f.TagsAnd == nil || len(f.TagsAnd) != 1 {
			return false
		}
		if !slices.Contains(f.TagsAnd["t"], "meme") || !slices.Contains(f.TagsAnd["t"], "cat") {
			return false
		}
		if f.Tags == nil || len(f.Tags) != 1 {
			return false
		}
		if !slices.Contains(f.Tags["t"], "black") || !slices.Contains(f.Tags["t"], "white") {
			return false
		}
		return true
	}, "failed to parse AND filter correctly")
}

func TestFilterMarshalWithAndTags(t *testing.T) {
	filterj, err := json.Marshal(Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme", "cat"}},
		Tags:    TagMap{"t": {"black", "white"}},
	})
	assert.NoError(t, err)

	// The order might vary, so we check that both &t and #t are present
	jsonStr := string(filterj)
	assert.Contains(t, jsonStr, `"&t"`)
	assert.Contains(t, jsonStr, `"#t"`)
	assert.Contains(t, jsonStr, `"meme"`)
	assert.Contains(t, jsonStr, `"cat"`)
	assert.Contains(t, jsonStr, `"black"`)
	assert.Contains(t, jsonStr, `"white"`)
}

func TestFilterMatchingWithAndTags(t *testing.T) {
	// Test: Event must have both "meme" AND "cat" tags
	filter := Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme", "cat"}},
	}

	// Event with both tags - should match
	event1 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
		},
	}
	assert.True(t, filter.Matches(event1), "event with both AND tags should match")

	// Event with only one tag - should not match
	event2 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
		},
	}
	assert.False(t, filter.Matches(event2), "event with only one AND tag should not match")

	// Event with neither tag - should not match
	event3 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "other"},
		},
	}
	assert.False(t, filter.Matches(event3), "event without AND tags should not match")
}

func TestFilterMatchingWithAndAndOrTags(t *testing.T) {
	// Test the example from the spec:
	// {"kinds": [1], "&t": ["meme", "cat"], "#t": ["black", "white"]}
	// Should match events with BOTH "meme" AND "cat" AND at least one of "black" OR "white"
	filter := Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme", "cat"}},
		Tags:    TagMap{"t": {"black", "white"}},
	}

	// Event with meme, cat, and black - should match
	event1 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
			Tag{"t", "black"},
		},
	}
	assert.True(t, filter.Matches(event1), "event with all required tags should match")

	// Event with meme, cat, and white - should match
	event2 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
			Tag{"t", "white"},
		},
	}
	assert.True(t, filter.Matches(event2), "event with meme, cat, and white should match")

	// Event with meme and cat but no black/white - should not match
	event3 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
		},
	}
	assert.False(t, filter.Matches(event3), "event missing OR tag should not match")

	// Event with meme, cat, black, but "meme" and "cat" should be ignored in OR evaluation
	// This tests that AND values are excluded from OR
	event4 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
			Tag{"t", "black"},
		},
	}
	assert.True(t, filter.Matches(event4), "event with AND tags and OR tag should match (AND values excluded from OR)")

	// Event with only meme (missing cat) - should not match even if it has black
	event5 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "black"},
		},
	}
	assert.False(t, filter.Matches(event5), "event missing one AND tag should not match")
}

func TestFilterMatchingAndTagsExcludedFromOr(t *testing.T) {
	// Test that values in AND are excluded from OR evaluation
	// If &t: ["meme"] and #t: ["meme", "other"], then "meme" should be ignored in OR
	filter := Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme"}},
		Tags:    TagMap{"t": {"meme", "other"}},
	}

	// Event with only "meme" - should NOT match because "other" is still required by OR
	event1 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
		},
	}
	assert.False(t, filter.Matches(event1), "event with only AND value should not match (OR still requires 'other')")

	// Event with "meme" and "other" - should match
	event2 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "other"},
		},
	}
	assert.True(t, filter.Matches(event2), "event with AND and OR values should match")

	// Event with only "other" (missing "meme") - should not match
	event3 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "other"},
		},
	}
	assert.False(t, filter.Matches(event3), "event missing AND value should not match")

	// Test case where all OR values are in AND - should match if AND is satisfied
	filter2 := Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme", "cat"}},
		Tags:    TagMap{"t": {"meme", "cat"}},
	}
	event4 := &Event{
		Kind: 1,
		Tags: Tags{
			Tag{"t", "meme"},
			Tag{"t", "cat"},
		},
	}
	assert.True(t, filter2.Matches(event4), "event with all AND values should match when all OR values are in AND")
}

func TestFilterEqualityWithAndTags(t *testing.T) {
	assert.True(t, FilterEqual(
		Filter{Kinds: []int{1}, TagsAnd: TagMap{"t": {"meme", "cat"}}},
		Filter{Kinds: []int{1}, TagsAnd: TagMap{"t": {"cat", "meme"}}},
	), "filters with same AND tags in different order should be equal")

	assert.False(t, FilterEqual(
		Filter{Kinds: []int{1}, TagsAnd: TagMap{"t": {"meme", "cat"}}},
		Filter{Kinds: []int{1}, TagsAnd: TagMap{"t": {"meme"}}},
	), "filters with different AND tags should not be equal")

	assert.True(t, FilterEqual(
		Filter{
			Kinds:   []int{1},
			TagsAnd: TagMap{"t": {"meme", "cat"}},
			Tags:    TagMap{"t": {"black", "white"}},
		},
		Filter{
			Kinds:   []int{1},
			TagsAnd: TagMap{"t": {"cat", "meme"}},
			Tags:    TagMap{"t": {"white", "black"}},
		},
	), "filters with same AND and OR tags should be equal")
}

func TestFilterCloneWithAndTags(t *testing.T) {
	flt := Filter{
		Kinds:   []int{1},
		TagsAnd: TagMap{"t": {"meme", "cat"}},
		Tags:    TagMap{"t": {"black", "white"}},
	}
	clone := flt.Clone()
	assert.True(t, FilterEqual(flt, clone), "clone with AND tags should be equal")

	clone1 := flt.Clone()
	clone1.TagsAnd["t"] = append(clone1.TagsAnd["t"], "dog")
	assert.False(t, FilterEqual(flt, clone1), "modifying clone AND tags should cause inequality")

	clone2 := flt.Clone()
	clone2.TagsAnd["new"] = []string{"value"}
	assert.False(t, FilterEqual(flt, clone2), "adding new AND tag to clone should cause inequality")
}
