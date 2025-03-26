package nip11

import (
	"slices"
)

type RelayInformationDocument struct {
	URL string `json:"-"`

	Name          string `json:"name"`
	Description   string `json:"description"`
	PubKey        string `json:"pubkey"`
	Contact       string `json:"contact"`
	SupportedNIPs []any  `json:"supported_nips"`
	Software      string `json:"software"`
	Version       string `json:"version"`

	Limitation     *RelayLimitationDocument  `json:"limitation,omitempty"`
	RelayCountries []string                  `json:"relay_countries,omitempty"`
	LanguageTags   []string                  `json:"language_tags,omitempty"`
	Tags           []string                  `json:"tags,omitempty"`
	PostingPolicy  string                    `json:"posting_policy,omitempty"`
	PaymentsURL    string                    `json:"payments_url,omitempty"`
	Fees           *RelayFeesDocument        `json:"fees,omitempty"`
	Retention      []*RelayRetentionDocument `json:"retention,omitempty"`
	Icon           string                    `json:"icon"`
	Banner         string                    `json:"banner"`
}

func (info *RelayInformationDocument) AddSupportedNIP(number int) {
	idx := slices.IndexFunc(info.SupportedNIPs, func(n any) bool { return n == number })
	if idx != -1 {
		return
	}

	info.SupportedNIPs = append(info.SupportedNIPs, number)
}

func (info *RelayInformationDocument) AddSupportedNIPs(numbers []int) {
	for _, n := range numbers {
		info.AddSupportedNIP(n)
	}
}

type RelayLimitationDocument struct {
	MaxMessageLength    int   `json:"max_message_length,omitempty"`
	MaxSubscriptions    int   `json:"max_subscriptions,omitempty"`
	MaxLimit            int   `json:"max_limit,omitempty"`
	DefaultLimit        int   `json:"default_limit,omitempty"`
	MaxSubidLength      int   `json:"max_subid_length,omitempty"`
	MaxEventTags        int   `json:"max_event_tags,omitempty"`
	MaxContentLength    int   `json:"max_content_length,omitempty"`
	MinPowDifficulty    int   `json:"min_pow_difficulty,omitempty"`
	CreatedAtLowerLimit int64 `json:"created_at_lower_limit"`
	CreatedAtUpperLimit int64 `json:"created_at_upper_limit"`
	AuthRequired        bool  `json:"auth_required"`
	PaymentRequired     bool  `json:"payment_required"`
	RestrictedWrites    bool  `json:"restricted_writes"`
}

type RelayFeesDocument struct {
	Admission []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"admission,omitempty"`
	Subscription []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
		Period int    `json:"period"`
	} `json:"subscription,omitempty"`
	Publication []struct {
		Kinds  []int  `json:"kinds"`
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"publication,omitempty"`
}

type RelayRetentionDocument struct {
	Time  int64   `json:"time,omitempty"`
	Count int     `json:"count,omitempty"`
	Kinds [][]int `json:"kinds,omitempty"`
}
