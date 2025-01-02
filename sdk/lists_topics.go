package sdk

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

type Topic string

func (r Topic) Value() string { return string(r) }

func (sys *System) FetchTopicList(ctx context.Context, pubkey string) GenericList[Topic] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10015, kind_10015, parseTopicString, sys.TopicListCache, false)
	return ml
}

func (sys *System) FetchTopicSets(ctx context.Context, pubkey string) GenericSets[Topic] {
	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30015, kind_30015, parseTopicString, sys.TopicSetsCache, false)
	return ml
}

func parseTopicString(tag nostr.Tag) (t Topic, ok bool) {
	if t := tag.Value(); t != "" && tag[0] == "t" {
		return Topic(t), true
	}
	return t, false
}
