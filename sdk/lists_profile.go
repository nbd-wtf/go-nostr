package sdk

import (
	"context"
	"net/url"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type ProfileRef struct {
	Pubkey  string
	Relay   string
	Petname string
}

func (f ProfileRef) Value() string { return f.Pubkey }

func (sys *System) FetchFollowList(ctx context.Context, pubkey string) GenericList[ProfileRef] {
	fl, _ := fetchGenericList(sys, ctx, pubkey, 3, kind_3, parseProfileRef, sys.FollowListCache, false)
	return fl
}

func (sys *System) FetchMuteList(ctx context.Context, pubkey string) GenericList[ProfileRef] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10000, kind_10000, parseProfileRef, sys.MuteListCache, false)
	return ml
}

func parseProfileRef(tag nostr.Tag) (fw ProfileRef, ok bool) {
	if len(tag) < 2 {
		return fw, false
	}
	if tag[0] != "p" {
		return fw, false
	}

	fw.Pubkey = tag[1]
	if !nostr.IsValidPublicKey(fw.Pubkey) {
		return fw, false
	}

	if len(tag) > 2 {
		if _, err := url.Parse(tag[2]); err == nil {
			fw.Relay = nostr.NormalizeURL(tag[2])
		}
		if len(tag) > 3 {
			fw.Petname = strings.TrimSpace(tag[3])
		}
		return fw, true
	}

	return fw, false
}
