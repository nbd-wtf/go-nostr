package nostr

import (
	"time"
)

type Filters []Filter

type Filter struct {
	IDs     StringList
	Kinds   IntList
	Authors StringList
	Since   *time.Time
	Until   *time.Time
	Tags    TagMap
}

type TagMap map[string]StringList

func (eff Filters) Match(event *Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (ef Filter) Matches(event *Event) bool {
	if event == nil {
		return false
	}

	if ef.IDs != nil && !ef.IDs.ContainsPrefixOf(event.ID) {
		return false
	}

	if ef.Kinds != nil && !ef.Kinds.Contains(event.Kind) {
		return false
	}

	if ef.Authors != nil && !ef.Authors.ContainsPrefixOf(event.PubKey) {
		return false
	}

	for f, v := range ef.Tags {
		if v != nil && !event.Tags.ContainsAny(f, v) {
			return false
		}
	}

	if ef.Since != nil && time.Time(event.CreatedAt).Before(*ef.Since) {
		return false
	}

	if ef.Until != nil && time.Time(event.CreatedAt).After(*ef.Until) {
		return false
	}

	return true
}

func FilterEqual(a Filter, b Filter) bool {
	if !a.Kinds.Equals(b.Kinds) {
		return false
	}

	if !a.IDs.Equals(b.IDs) {
		return false
	}

	if !a.Authors.Equals(b.Authors) {
		return false
	}

	if len(a.Tags) != len(b.Tags) {
		return false
	}

	for f, av := range a.Tags {
		if bv, ok := b.Tags[f]; !ok {
			return false
		} else {
			if !av.Equals(bv) {
				return false
			}
		}
	}

	if a.Since != b.Since {
		return false
	}

	if a.Until != b.Until {
		return false
	}

	return true
}
