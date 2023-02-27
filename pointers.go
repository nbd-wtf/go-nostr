package nostr

type ProfilePointer struct {
	PublicKey string
	Relays    []string
}

type EventPointer struct {
	ID     string
	Relays []string
}

type EntityPointer struct {
	PublicKey  string
	Kind       int
	Identifier string
	Relays     []string
}
