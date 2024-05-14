package nip52

import (
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type CalendarEventKind int

const (
	TimeBased = 31923
	DateBased = 31922
)

type CalendarEvent struct {
	CalendarEventKind
	Identifier   string
	Title        string
	Image        string
	Start, End   time.Time
	Locations    []string
	Geohashes    []string
	Participants []Participant
	References   []string
	Hashtags     []string
	StartTzid    string
	EndTzid      string
}

type Participant struct {
	PubKey string
	Relay  string
	Role   string
}

func ParseCalendarEvent(event nostr.Event) CalendarEvent {
	calev := CalendarEvent{
		CalendarEventKind: CalendarEventKind(event.Kind),
	}
	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			calev.Identifier = tag[1]
		case "title":
			calev.Title = tag[1]
		case "image":
			calev.Image = tag[1]
		case "start", "end":
			var v time.Time
			switch calev.CalendarEventKind {
			case TimeBased:
				i, err := strconv.ParseInt(tag[1], 10, 64)
				if err != nil {
					continue
				}
				v = time.Unix(i, 0)
			case DateBased:
				var err error
				v, err = time.Parse(DateFormat, tag[1])
				if err != nil {
					continue
				}
			}
			switch tag[0] {
			case "start":
				calev.Start = v
			case "end":
				calev.End = v
			}
		case "location":
			calev.Locations = append(calev.Locations, tag[1])
		case "g":
			calev.Geohashes = append(calev.Geohashes, tag[1])
		case "p":
			if nostr.IsValid32ByteHex(tag[1]) {
				part := Participant{
					PubKey: tag[1],
				}
				if len(tag) > 2 {
					part.Relay = tag[2]
					if len(tag) > 3 {
						part.Role = tag[3]
					}
				}
				calev.Participants = append(calev.Participants, part)
			}
		case "r":
			calev.References = append(calev.References, tag[1])
		case "t":
			calev.Hashtags = append(calev.Hashtags, tag[1])
		case "start_tzid":
			calev.StartTzid = tag[1]
		case "end_tzid":
			calev.EndTzid = tag[1]
		}
	}
	return calev
}

func (calev CalendarEvent) ToHashtags() nostr.Tags {
	tags := make(nostr.Tags, 0, 26)
	tags = append(tags, nostr.Tag{"d", calev.Identifier})
	tags = append(tags, nostr.Tag{"title", calev.Title})
	if calev.Image != "" {
		tags = append(tags, nostr.Tag{"image", calev.Title})
	}

	if calev.CalendarEventKind == TimeBased {
		tags = append(tags, nostr.Tag{"start", strconv.FormatInt(calev.Start.Unix(), 10)})
		if !calev.End.IsZero() {
			tags = append(tags, nostr.Tag{"end", strconv.FormatInt(calev.End.Unix(), 10)})
		}
	} else if calev.CalendarEventKind == DateBased {
		tags = append(tags, nostr.Tag{"start", calev.Start.Format(DateFormat)})
		if !calev.End.IsZero() {
			tags = append(tags, nostr.Tag{"end", calev.End.Format(DateFormat)})
		}
	}

	for _, location := range calev.Locations {
		tags = append(tags, nostr.Tag{"location", location})
	}
	for _, geohash := range calev.Geohashes {
		tags = append(tags, nostr.Tag{"g", geohash})
	}
	for _, part := range calev.Participants {
		tags = append(tags, nostr.Tag{"p", part.PubKey, part.Relay, part.Role})
	}
	for _, reference := range calev.References {
		tags = append(tags, nostr.Tag{"r", reference})
	}
	for _, hashtag := range calev.Hashtags {
		tags = append(tags, nostr.Tag{"t", hashtag})
	}

	return tags
}
