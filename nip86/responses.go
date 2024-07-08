package nip86

type IDReason struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type PubKeyReason struct {
	PubKey string `json:"pubkey"`
	Reason string `json:"reason"`
}

type IPReason struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}
