package nip34

import "github.com/nbd-wtf/go-nostr"

type Repository struct {
	nostr.Event

	ID          string
	Name        string
	Description string
	Web         []string
	Clone       []string
	Relays      []string
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
			repo.Web = append(repo.Web, tag[1])
		case "clone":
			repo.Clone = append(repo.Clone, tag[1])
		case "relays":
			repo.Relays = append(repo.Relays, tag[1])
		}
	}

	return repo
}
