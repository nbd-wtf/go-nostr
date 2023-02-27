package nostr

import (
	"encoding/json"
	"time"

	"golang.org/x/exp/slices"
)

type Filters []Filter

type Filter struct {
	IDs     []string
	Kinds   []int
	Authors []string
	Tags    TagMap
	Since   *time.Time
	Until   *time.Time
	Limit   int
	Search  string
}

type TagMap map[string][]string

func (eff Filters) String() string {
	j, _ := json.Marshal(eff)
	return string(j)
}

func (eff Filters) Match(event *Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (ef Filter) String() string {
	j, _ := json.Marshal(ef)
	return string(j)
}

func (ef Filter) Matches(event *Event) bool {
	if event == nil {
		return false
	}

	if ef.IDs != nil && !containsPrefixOf(ef.IDs, event.ID) {
		return false
	}

	if ef.Kinds != nil && !slices.Contains(ef.Kinds, event.Kind) {
		return false
	}

	if ef.Authors != nil && !containsPrefixOf(ef.Authors, event.PubKey) {
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
	if !similar(a.Kinds, b.Kinds) {
		return false
	}

	if !similar(a.IDs, b.IDs) {
		return false
	}

	if !similar(a.Authors, b.Authors) {
		return false
	}

	if len(a.Tags) != len(b.Tags) {
		return false
	}

	for f, av := range a.Tags {
		if bv, ok := b.Tags[f]; !ok {
			return false
		} else {
			if !similar(av, bv) {
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

	if a.Search != b.Search {
		return false
	}

	return true
}
