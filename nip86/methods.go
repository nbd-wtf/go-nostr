package nip86

import (
	"fmt"
	"math"
	"net"

	"github.com/nbd-wtf/go-nostr"
)

func DecodeRequest(req Request) (MethodParams, error) {
	switch req.Method {
	case "supportedmethods":
		return SupportedMethods{}, nil
	case "banpubkey":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		pk, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKey(pk) {
			return nil, fmt.Errorf("invalid pubkey param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return BanPubKey{pk, reason}, nil
	case "listbannedpubkeys":
		return ListBannedPubKeys{}, nil
	case "allowpubkey":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		pk, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKey(pk) {
			return nil, fmt.Errorf("invalid pubkey param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return AllowPubKey{pk, reason}, nil
	case "listallowedpubkeys":
		return ListAllowedPubKeys{}, nil
	case "listeventsneedingmoderation":
		return ListEventsNeedingModeration{}, nil
	case "allowevent":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		id, ok := req.Params[0].(string)
		if !ok || !nostr.IsValid32ByteHex(id) {
			return nil, fmt.Errorf("invalid id param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return AllowEvent{id, reason}, nil
	case "banevent":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		id, ok := req.Params[0].(string)
		if !ok || !nostr.IsValid32ByteHex(id) {
			return nil, fmt.Errorf("invalid id param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return BanEvent{id, reason}, nil
	case "listbannedevents":
		return ListBannedEvents{}, nil
	case "changerelayname":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		name, _ := req.Params[0].(string)
		return ChangeRelayName{name}, nil
	case "changerelaydescription":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		desc, _ := req.Params[0].(string)
		return ChangeRelayDescription{desc}, nil
	case "changerelayicon":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		url, _ := req.Params[0].(string)
		return ChangeRelayIcon{url}, nil
	case "allowkind":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		kind, ok := req.Params[0].(float64)
		if !ok || math.Trunc(kind) != kind {
			return nil, fmt.Errorf("invalid kind '%v' for '%s'", req.Params[0], req.Method)
		}
		return AllowKind{nostr.Kind(kind)}, nil
	case "disallowkind":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		kind, ok := req.Params[0].(float64)
		if !ok || math.Trunc(kind) != kind {
			return nil, fmt.Errorf("invalid kind '%v' for '%s'", req.Params[0], req.Method)
		}
		return DisallowKind{nostr.Kind(kind)}, nil
	case "listallowedkinds":
		return ListAllowedKinds{}, nil
	case "blockip":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		ipstr, _ := req.Params[0].(string)
		ip := net.ParseIP(ipstr)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return BlockIP{ip, reason}, nil
	case "unblockip":
		if len(req.Params) == 0 {
			return nil, fmt.Errorf("invalid number of params for '%s'", req.Method)
		}
		ipstr, _ := req.Params[0].(string)
		ip := net.ParseIP(ipstr)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip param for '%s'", req.Method)
		}
		var reason string
		if len(req.Params) >= 2 {
			reason, _ = req.Params[1].(string)
		}
		return UnblockIP{ip, reason}, nil
	case "listblockedips":
		return ListBlockedIPs{}, nil
	default:
		return nil, fmt.Errorf("unknown method '%s'", req.Method)
	}
}

type MethodParams interface {
	MethodName() string
}

var (
	_ MethodParams = (*SupportedMethods)(nil)
	_ MethodParams = (*BanPubKey)(nil)
	_ MethodParams = (*ListBannedPubKeys)(nil)
	_ MethodParams = (*AllowPubKey)(nil)
	_ MethodParams = (*ListAllowedPubKeys)(nil)
	_ MethodParams = (*ListEventsNeedingModeration)(nil)
	_ MethodParams = (*AllowEvent)(nil)
	_ MethodParams = (*BanEvent)(nil)
	_ MethodParams = (*ListBannedEvents)(nil)
	_ MethodParams = (*ChangeRelayName)(nil)
	_ MethodParams = (*ChangeRelayDescription)(nil)
	_ MethodParams = (*ChangeRelayIcon)(nil)
	_ MethodParams = (*AllowKind)(nil)
	_ MethodParams = (*DisallowKind)(nil)
	_ MethodParams = (*ListAllowedKinds)(nil)
	_ MethodParams = (*BlockIP)(nil)
	_ MethodParams = (*UnblockIP)(nil)
	_ MethodParams = (*ListBlockedIPs)(nil)
)

type SupportedMethods struct{}

func (_ SupportedMethods) MethodName() string { return "supportedmethods" }

type BanPubKey struct {
	PubKey string
	Reason string
}

func (_ BanPubKey) MethodName() string { return "banpubkey" }

type ListBannedPubKeys struct{}

func (_ ListBannedPubKeys) MethodName() string { return "listbannedpubkeys" }

type AllowPubKey struct {
	PubKey string
	Reason string
}

func (_ AllowPubKey) MethodName() string { return "allowpubkey" }

type ListAllowedPubKeys struct{}

func (_ ListAllowedPubKeys) MethodName() string { return "listallowedpubkeys" }

type ListEventsNeedingModeration struct{}

func (_ ListEventsNeedingModeration) MethodName() string { return "listeventsneedingmoderation" }

type AllowEvent struct {
	ID     string
	Reason string
}

func (_ AllowEvent) MethodName() string { return "allowevent" }

type BanEvent struct {
	ID     string
	Reason string
}

func (_ BanEvent) MethodName() string { return "banevent" }

type ListBannedEvents struct{}

func (_ ListBannedEvents) MethodName() string { return "listbannedevents" }

type ChangeRelayName struct {
	Name string
}

func (_ ChangeRelayName) MethodName() string { return "changerelayname" }

type ChangeRelayDescription struct {
	Description string
}

func (_ ChangeRelayDescription) MethodName() string { return "changerelaydescription" }

type ChangeRelayIcon struct {
	IconURL string
}

func (_ ChangeRelayIcon) MethodName() string { return "changerelayicon" }

type AllowKind struct {
	Kind nostr.Kind
}

func (_ AllowKind) MethodName() string { return "allowkind" }

type DisallowKind struct {
	Kind nostr.Kind
}

func (_ DisallowKind) MethodName() string { return "disallowkind" }

type ListAllowedKinds struct{}

func (_ ListAllowedKinds) MethodName() string { return "listallowedkinds" }

type BlockIP struct {
	IP     net.IP
	Reason string
}

func (_ BlockIP) MethodName() string { return "blockip" }

type UnblockIP struct {
	IP     net.IP
	Reason string
}

func (_ UnblockIP) MethodName() string { return "unblockip" }

type ListBlockedIPs struct{}

func (_ ListBlockedIPs) MethodName() string { return "listblockedips" }
