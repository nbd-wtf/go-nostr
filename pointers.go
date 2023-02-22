package nostr

type ProfilePointer struct {
	PublicKey string
	Relays    []string
}

type EventPointer struct {
	ID     string
	Relays []string
}
