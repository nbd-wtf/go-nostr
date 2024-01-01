package nostr

type Kind uint16

const (
	KindProfileMetadata          Kind = 0
	KindTextNote                 Kind = 1
	KindRecommendServer          Kind = 2
	KindContactList              Kind = 3
	KindEncryptedDirectMessage   Kind = 4
	KindDeletion                 Kind = 5
	KindRepost                   Kind = 6
	KindReaction                 Kind = 7
	KindChannelCreation          Kind = 40
	KindChannelMetadata          Kind = 41
	KindChannelMessage           Kind = 42
	KindChannelHideMessage       Kind = 43
	KindChannelMuteUser          Kind = 44
	KindFileMetadata             Kind = 1063
	KindZapRequest               Kind = 9734
	KindZap                      Kind = 9735
	KindMuteList                 Kind = 10000
	KindPinList                  Kind = 10001
	KindRelayListMetadata        Kind = 10002
	KindNWCWalletInfo            Kind = 13194
	KindClientAuthentication     Kind = 22242
	KindNWCWalletRequest         Kind = 23194
	KindNWCWalletResponse        Kind = 23195
	KindNostrConnect             Kind = 24133
	KindCategorizedPeopleList    Kind = 30000
	KindCategorizedBookmarksList Kind = 30001
	KindProfileBadges            Kind = 30008
	KindBadgeDefinition          Kind = 30009
	KindStallDefinition          Kind = 30017
	KindProductDefinition        Kind = 30018
	KindArticle                  Kind = 30023
	KindApplicationSpecificData  Kind = 30078
)
