package nip11

type RelayInformationDocument struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PubKey        string `json:"pubkey"`
	Contact       string `json:"contact"`
	SupportedNIPs []int  `json:"supported_nips"`
	Software      string `json:"software"`
	Version       string `json:"version"`
}
