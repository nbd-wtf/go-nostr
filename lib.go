package nostr

import (
	"github.com/nbd-wtf/go-nostr/core"
	"github.com/nbd-wtf/go-nostr/relays"
	"github.com/nbd-wtf/go-nostr/utils"
)

type (
	Event     core.Event
	Filter    core.Filter
	Filters   core.Filters
	Timestamp core.Timestamp
	Tag       core.Tag
	TagMap    core.TagMap

	ProfilePointer core.ProfilePointer
	EventPointer   core.EventPointer
	EntityPointer  core.EntityPointer

	AuthEnvelope   core.AuthEnvelope
	OKEnvelope     core.OKEnvelope
	NoticeEnvelope core.NoticeEnvelope
	EventEnvelope  core.EventEnvelope
	CloseEnvelope  core.CloseEnvelope
	ClosedEnvelope core.ClosedEnvelope
	CountEnvelope  core.CountEnvelope
	EOSEEnvelope   core.EOSEEnvelope
	ReqEnvelope    core.ReqEnvelope
	Envelope       core.Envelope

	Relay              relays.Relay
	RelayOption        relays.RelayOption
	SimplePool         relays.SimplePool
	PoolOption         relays.PoolOption
	Subscription       relays.Subscription
	SubscriptionOption relays.SubscriptionOption
	IncomingEvent      relays.IncomingEvent
	WithAuthHandler    relays.WithAuthHandler
	WithLabel          relays.WithLabel
	WithNoticeHandler  relays.WithNoticeHandler

	RelayStore relays.RelayStore
	MultiStore relays.MultiStore
)

var (
	Now                = core.Now
	FilterEqual        = core.FilterEqual
	GeneratePrivateKey = core.GeneratePrivateKey
	GetPublicKey       = core.GetPublicKey
	IsValidPublicKey   = core.IsValidPublicKey

	NewSimplePool = relays.NewSimplePool
	NewRelay      = relays.NewRelay

	IsValid32ByteHex   = utils.IsValid32ByteHex
	IsValidRelayURL    = utils.IsValidRelayURL
	NormalizeOKMessage = utils.NormalizeOKMessage
	NormalizeURL       = utils.NormalizeURL
)
