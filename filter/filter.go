package filter

import "github.com/fiatjaf/go-nostr/event"

type EventFilters []EventFilter

type EventFilter struct {
	IDs     []string `json:"ids,omitempty"`
	Kinds   []int    `json:"kinds,omitempty"`
	Authors []string `json:"authors,omitempty"`
	Since   uint32   `json:"since,omitempty"`
	Until   uint32   `json:"until,omitempty"`
	TagE    []string `json:"#e,omitempty"`
	TagP    []string `json:"#p,omitempty"`
}

func (eff EventFilters) Match(event *event.Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (ef EventFilter) Matches(event *event.Event) bool {
	if event == nil {
		return false
	}

	if ef.IDs != nil && !stringsContain(ef.IDs, event.ID) {
		return false
	}

	if ef.Kinds != nil && !intsContain(ef.Kinds, event.Kind) {
		return false
	}

	if ef.Authors != nil && !stringsContain(ef.Authors, event.PubKey) {
		return false
	}

	if ef.TagE != nil && !containsAnyTag("e", event.Tags, ef.TagE) {
		return false
	}

	if ef.TagP != nil && !containsAnyTag("p", event.Tags, ef.TagP) {
		return false
	}

	if ef.Since != 0 && event.CreatedAt < ef.Since {
		return false
	}

	if ef.Until != 0 && event.CreatedAt >= ef.Until {
		return false
	}

	return true
}

func Equal(a EventFilter, b EventFilter) bool {
	if !intsEqual(a.Kinds, b.Kinds) {
		return false
	}

	if !stringsEqual(a.IDs, b.IDs) {
		return false
	}

	if !stringsEqual(a.Authors, b.Authors) {
		return false
	}

	if !stringsEqual(a.TagE, b.TagE) {
		return false
	}

	if !stringsEqual(a.TagP, b.TagP) {
		return false
	}

	if a.Since != b.Since {
		return false
	}

	if a.Until != b.Until {
		return false
	}

	return true
}
