package filter

import "github.com/fiatjaf/go-nostr/event"

type EventFilter struct {
	ID         string   `json:"id,omitempty"`
	Author     string   `json:"author,omitempty"`
	Kind       *int     `json:"kind,omitempty"`
	Authors    []string `json:"authors,omitempty"`
	TagEvent   string   `json:"#e,omitempty"`
	TagProfile string   `json:"#p,omitempty"`
	Since      uint32   `json:"since,omitempty"`
}

func (ef EventFilter) Matches(event *event.Event) bool {
	if event == nil {
		return false
	}

	if ef.ID != "" && ef.ID != event.ID {
		return false
	}

	if ef.Author != "" && ef.Author != event.PubKey {
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

	return true
}
