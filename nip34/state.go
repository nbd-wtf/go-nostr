package nip34

import (
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type RepositoryState struct {
	nostr.Event

	ID       string
	HEAD     string
	Tags     map[string]string
	Branches map[string]string
}

func ParseRepositoryState(event nostr.Event) RepositoryState {
	st := RepositoryState{
		Event:    event,
		Tags:     make(map[string]string),
		Branches: make(map[string]string),
	}

	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			st.ID = tag[1]
		case "HEAD":
			if strings.HasPrefix(tag[1], "ref: refs/heads/") {
				st.HEAD = tag[1][16:]
			}
		default:
			if strings.HasPrefix(tag[0], "refs/heads/") {
				st.Branches[tag[0][11:]] = tag[1]
			} else if strings.HasPrefix(tag[0], "refs/tags/") {
				st.Tags[tag[0][10:]] = tag[1]
			}
		}
	}

	return st
}

func (rs RepositoryState) ToEvent() *nostr.Event {
	tags := make(nostr.Tags, 1, 2+len(rs.Branches)+len(rs.Tags))

	tags[0] = nostr.Tag{"d", rs.ID}
	for branchName, commitId := range rs.Branches {
		tags = append(tags, nostr.Tag{"refs/heads/" + branchName, commitId})
	}
	for tagName, commitId := range rs.Tags {
		tags = append(tags, nostr.Tag{"refs/tags/" + tagName, commitId})
	}
	if rs.HEAD != "" {
		tags = append(tags, nostr.Tag{"HEAD", "ref: refs/heads/" + rs.HEAD})
	}

	return &nostr.Event{
		Kind:      30618,
		Tags:      tags,
		CreatedAt: nostr.Now(),
	}
}
