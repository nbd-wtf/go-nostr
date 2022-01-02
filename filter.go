package nostr

type EventFilters []EventFilter

type EventFilter struct {
	IDs     StringList `json:"ids,omitempty"`
	Kinds   IntList    `json:"kinds,omitempty"`
	Authors StringList `json:"authors,omitempty"`
	Since   uint32     `json:"since,omitempty"`
	Until   uint32     `json:"until,omitempty"`
	TagE    StringList `json:"#e,omitempty"`
	TagP    StringList `json:"#p,omitempty"`
}

func (eff EventFilters) Match(event *Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (ef EventFilter) Matches(event *Event) bool {
	if event == nil {
		return false
	}

	if ef.IDs != nil && !ef.IDs.Contains(event.ID) {
		return false
	}

	if ef.Kinds != nil && !ef.Kinds.Contains(event.Kind) {
		return false
	}

	if ef.Authors != nil && !ef.Authors.Contains(event.PubKey) {
		return false
	}

	if ef.TagE != nil && !event.Tags.ContainsAny("e", ef.TagE) {
		return false
	}

	if ef.TagP != nil && !event.Tags.ContainsAny("p", ef.TagP) {
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
	if !a.Kinds.Equals(b.Kinds) {
		return false
	}

	if !a.IDs.Equals(b.IDs) {
		return false
	}

	if !a.Authors.Equals(b.Authors) {
		return false
	}

	if !a.TagE.Equals(b.TagE) {
		return false
	}

	if !a.TagP.Equals(b.TagP) {
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
