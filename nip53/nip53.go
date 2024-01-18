package nip53

import (
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type LiveEvent struct {
	Identifier          string
	Title               string
	Summary             string
	Image               string
	Status              string
	CurrentParticipants int
	TotalParticipants   int
	Streaming           []string
	Recording           []string
	Starts, Ends        time.Time
	Participants        []Participant
	Hashtags            []string
	Relays              []string
}

type Participant struct {
	PubKey string
	Relay  string
	Role   string
}

func ParseLiveEvent(event nostr.Event) LiveEvent {
	liev := LiveEvent{}
	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			liev.Identifier = tag[1]
		case "title":
			liev.Title = tag[1]
		case "summary":
			liev.Summary = tag[1]
		case "image":
			liev.Image = tag[1]
		case "status":
			liev.Status = tag[1]
		case "starts", "ends":
			var v time.Time
			i, err := strconv.ParseInt(tag[1], 10, 64)
			if err != nil {
				continue
			}
			v = time.Unix(i, 0)
			switch tag[0] {
			case "start":
				liev.Starts = v
			case "end":
				liev.Ends = v
			}
		case "streaming":
			liev.Streaming = append(liev.Streaming, tag[1])
		case "recording":
			liev.Recording = append(liev.Recording, tag[1])
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
				liev.Participants = append(liev.Participants, part)
			}
		case "relays":
			liev.Relays = append(liev.Relays, tag[1])
		case "t":
			liev.Hashtags = append(liev.Hashtags, tag[1])
		case "current_participants":
			liev.CurrentParticipants, _ = strconv.Atoi(tag[1])
		case "total_participants":
			liev.TotalParticipants, _ = strconv.Atoi(tag[1])
		}
	}
	return liev
}

func (liev LiveEvent) GetHost() *Participant {
	for _, part := range liev.Participants {
		if part.Role == "host" {
			return &part
		}
	}
	return nil
}

func (liev LiveEvent) ToHashtags() nostr.Tags {
	tags := make(nostr.Tags, 0, 26)
	tags = append(tags, nostr.Tag{"d", liev.Identifier})
	tags = append(tags, nostr.Tag{"title", liev.Title})
	if liev.Image != "" {
		tags = append(tags, nostr.Tag{"image", liev.Title})
	}

	tags = append(tags, nostr.Tag{"start", strconv.FormatInt(liev.Starts.Unix(), 10)})
	if !liev.Ends.IsZero() {
		tags = append(tags, nostr.Tag{"end", strconv.FormatInt(liev.Ends.Unix(), 10)})
	}

	for _, url := range liev.Streaming {
		tags = append(tags, nostr.Tag{"streaming", url})
	}
	for _, url := range liev.Recording {
		tags = append(tags, nostr.Tag{"recording", url})
	}
	for _, part := range liev.Participants {
		tags = append(tags, nostr.Tag{"p", part.PubKey, part.Relay, part.Role})
	}
	for _, hashtag := range liev.Hashtags {
		tags = append(tags, nostr.Tag{"t", hashtag})
	}
	if liev.CurrentParticipants != 0 {
		tags = append(tags, nostr.Tag{"current_participants", strconv.Itoa(liev.CurrentParticipants)})
	}
	if liev.TotalParticipants != 0 {
		tags = append(tags, nostr.Tag{"total_participants", strconv.Itoa(liev.TotalParticipants)})
	}
	for _, r := range liev.Relays {
		tags = append(tags, nostr.Tag{"relay", r})
	}

	return tags
}
