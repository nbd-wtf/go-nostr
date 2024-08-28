package nip34

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

type Repository struct {
	nostr.Event

	ID                     string
	Name                   string
	Description            string
	Web                    []string
	Clone                  []string
	Relays                 []string
	EarliestUniqueCommitID string
	Maintainers            []string
}

func ParseRepository(event nostr.Event) Repository {
	repo := Repository{
		Event: event,
	}

	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			repo.ID = tag[1]
		case "name":
			repo.Name = tag[1]
		case "description":
			repo.Description = tag[1]
		case "web":
			repo.Web = append(repo.Web, tag[1:]...)
		case "clone":
			repo.Clone = append(repo.Clone, tag[1:]...)
		case "relays":
			repo.Relays = append(repo.Relays, tag[1:]...)
		case "r":
			repo.EarliestUniqueCommitID = tag[1]
		case "maintainers":
			repo.Maintainers = append(repo.Maintainers, tag[1:]...)
		}
	}

	return repo
}

func (r Repository) ToEvent() *nostr.Event {
	tags := make(nostr.Tags, 0, 10)

	tags = append(tags, nostr.Tag{"d", r.ID})

	if r.Name != "" {
		tags = append(tags, nostr.Tag{"name", r.Name})
	}
	if r.Description != "" {
		tags = append(tags, nostr.Tag{"description", r.Description})
	}
	if r.EarliestUniqueCommitID != "" {
		tags = append(tags, nostr.Tag{"r", r.EarliestUniqueCommitID, "euc"})
	}
	if len(r.Maintainers) > 0 {
		tag := make(nostr.Tag, 1, 1+len(r.Maintainers))
		tag[0] = "maintainers"
		tag = append(tag, r.Maintainers...)
		tags = append(tags, tag)
	}
	if len(r.Web) > 0 {
		tag := make(nostr.Tag, 1, 1+len(r.Web))
		tag[0] = "web"
		tag = append(tag, r.Web...)
		tags = append(tags, tag)
	}
	if len(r.Clone) > 0 {
		tag := make(nostr.Tag, 1, 1+len(r.Clone))
		tag[0] = "clone"
		tag = append(tag, r.Clone...)
		tags = append(tags, tag)
	}
	if len(r.Relays) > 0 {
		tag := make(nostr.Tag, 1, 1+len(r.Relays))
		tag[0] = "relays"
		tag = append(tag, r.Relays...)
		tags = append(tags, tag)
	}

	return &nostr.Event{
		Kind:      nostr.KindRepositoryAnnouncement,
		Tags:      tags,
		CreatedAt: nostr.Now(),
	}
}

func (repo Repository) FetchState(ctx context.Context, s nostr.RelayStore) *RepositoryState {
	res, _ := s.QuerySync(ctx, nostr.Filter{
		Kinds: []int{nostr.KindRepositoryState},
		Tags: nostr.TagMap{
			"d": []string{repo.Tags.GetD()},
		},
	})

	if len(res) == 0 {
		return nil
	}

	rs := ParseRepositoryState(*res[0])
	return &rs
}

func (repo Repository) GetPatchesSync(ctx context.Context, s nostr.RelayStore) []Patch {
	res, _ := s.QuerySync(ctx, nostr.Filter{
		Kinds: []int{nostr.KindPatch},
		Tags: nostr.TagMap{
			"a": []string{fmt.Sprintf("%d:%s:%s", nostr.KindRepositoryAnnouncement, repo.Event.PubKey, repo.ID)},
		},
	})
	patches := make([]Patch, len(res))
	for i, evt := range res {
		patches[i] = ParsePatch(*evt)
	}
	return patches
}
