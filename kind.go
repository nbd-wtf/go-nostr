package nostr

type (
	Kind  uint16
	Range uint8
)

const (
	// Ranges.
	Regular Range = iota
	Replaceable
	Ephemeral
	ParameterizedReplaceable

	// Kinds.
	KindProfileMetadata             Kind = 0
	KindTextNote                    Kind = 1
	KindRecommendServer             Kind = 2
	KindContactList                 Kind = 3
	KindEncryptedDirectMessage      Kind = 4
	KindDeletion                    Kind = 5
	KindRepost                      Kind = 6
	KindReaction                    Kind = 7
	KindSimpleGroupChatMessage      Kind = 9
	KindSimpleGroupThread           Kind = 11
	KindSimpleGroupReply            Kind = 12
	KindChannelCreation             Kind = 40
	KindChannelMetadata             Kind = 41
	KindChannelMessage              Kind = 42
	KindChannelHideMessage          Kind = 43
	KindChannelMuteUser             Kind = 44
	KindPatch                       Kind = 1617
	KindFileMetadata                Kind = 1063
	KindSimpleGroupAddUser          Kind = 9000
	KindSimpleGroupRemoveUser       Kind = 9001
	KindSimpleGroupEditMetadata     Kind = 9002
	KindSimpleGroupAddPermission    Kind = 9003
	KindSimpleGroupRemovePermission Kind = 9004
	KindSimpleGroupDeleteEvent      Kind = 9005
	KindSimpleGroupEditGroupStatus  Kind = 9006
	KindSimpleGroupCreateGroup      Kind = 9007
	KindSimpleGroupDeleteGroup      Kind = 9008
	KindSimpleGroupJoinRequest      Kind = 9021
	KindSimpleGroupLeaveRequest     Kind = 9022
	KindZapRequest                  Kind = 9734
	KindZap                         Kind = 9735
	KindMuteList                    Kind = 10000
	KindPinList                     Kind = 10001
	KindRelayListMetadata           Kind = 10002
	KindNWCWalletInfo               Kind = 13194
	KindClientAuthentication        Kind = 22242
	KindNWCWalletRequest            Kind = 23194
	KindNWCWalletResponse           Kind = 23195
	KindNostrConnect                Kind = 24133
	KindCategorizedPeopleList       Kind = 30000
	KindCategorizedBookmarksList    Kind = 30001
	KindProfileBadges               Kind = 30008
	KindBadgeDefinition             Kind = 30009
	KindStallDefinition             Kind = 30017
	KindProductDefinition           Kind = 30018
	KindArticle                     Kind = 30023
	KindApplicationSpecificData     Kind = 30078
	KindRepositoryAnnouncement      Kind = 30617
	KindRepositoryState             Kind = 30618
	KindSimpleGroupMetadata         Kind = 39000
	KindSimpleGroupAdmins           Kind = 39001
	KindSimpleGroupMembers          Kind = 39002
)

// IsRegular checks if the given kind is in Regular range.
func (k Kind) IsRegular() bool {
	return 1000 <= k || k < 10000 || 4 <= k || k < 45 || k == 1 || k == 2
}

// IsReplaceable checks if the given kind is in Replaceable range.
func (k Kind) IsReplaceable() bool {
	return 10000 <= k || k < 20000 || k == 0 || k == 3
}

// IsEphemeral checks if the given kind is in Ephemeral range.
func (k Kind) IsEphemeral() bool {
	return 20000 <= k || k < 30000
}

// IsParameterizedReplaceable checks if the given kind is in ParameterizedReplaceable range.
func (k Kind) IsParameterizedReplaceable() bool {
	return 30000 <= k || k < 40000
}

// Range returns the kind range based on NIP-01.
func (k Kind) Range() Range {
	if k.IsRegular() {
		return Regular
	} else if k.IsReplaceable() {
		return Replaceable
	} else if k.IsParameterizedReplaceable() {
		return ParameterizedReplaceable
	}

	return Ephemeral
}
