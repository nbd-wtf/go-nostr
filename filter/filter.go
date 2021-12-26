package filter

import "github.com/fiatjaf/go-nostr/event"

type EventFilters []EventFilter

type EventFilter struct {
	ID         string   `json:"id,omitempty"`
	Kind       *int     `json:"kind,omitempty"`
	Authors    []string `json:"authors,omitempty"`
	TagEvent   string   `json:"#e,omitempty"`
	TagProfile string   `json:"#p,omitempty"`
	Since      uint32   `json:"since,omitempty"`
	Until      uint32   `json:"until,omitempty"`
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

	if ef.ID != "" && ef.ID != event.ID {
		return false
	}

	if ef.Authors != nil {
		found := false
		for _, pubkey := range ef.Authors {
			if pubkey == event.PubKey {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if ef.TagEvent != "" {
		found := false
		for _, tag := range event.Tags {
			if len(tag) < 2 {
				continue
			}

			tagType, ok := tag[0].(string)
			if !ok {
				continue
			}

			if tagType == "e" {
				taggedID, ok := tag[1].(string)
				if !ok {
					continue
				}

				if taggedID == ef.TagEvent {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	if ef.TagProfile != "" {
		found := false
		for _, tag := range event.Tags {
			if len(tag) < 2 {
				continue
			}

			tagType, ok := tag[0].(string)
			if !ok {
				continue
			}

			if tagType == "p" {
				taggedID, ok := tag[1].(string)
				if !ok {
					continue
				}

				if taggedID == ef.TagProfile {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	if ef.Kind != nil && *ef.Kind != event.Kind {
		return false
	}

	if ef.Since != 0 && event.CreatedAt < ef.Since {
		return false
	}

	if ef.Until != 0 && event.CreatedAt > ef.Until {
		return false
	}

	return true
}

func Equal(a EventFilter, b EventFilter) bool {
	if a.Kind == nil && b.Kind != nil ||
		a.Kind != nil && b.Kind == nil ||
		a.Kind != b.Kind {
		return false
	}

	if a.ID != b.ID {
		return false
	}

	if len(a.Authors) != len(b.Authors) {
		return false
	}

	for i, _ := range a.Authors {
		if a.Authors[i] != b.Authors[i] {
			return false
		}
	}

	if a.TagEvent != b.TagEvent {
		return false
	}

	if a.TagProfile != b.TagProfile {
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
