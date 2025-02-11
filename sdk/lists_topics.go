package sdk

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	cache_memory "github.com/nbd-wtf/go-nostr/sdk/cache/memory"
)

type Topic string

func (r Topic) Value() string { return string(r) }

func (sys *System) FetchTopicList(ctx context.Context, pubkey string) GenericList[Topic] {
	if sys.TopicListCache == nil {
		sys.TopicListCache = cache_memory.New32[GenericList[Topic]](1000)
	}

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10015, kind_10015, parseTopicString, sys.TopicListCache)
	return ml
}

func (sys *System) FetchTopicSets(ctx context.Context, pubkey string) GenericSets[Topic] {
	if sys.TopicSetsCache == nil {
		sys.TopicSetsCache = cache_memory.New32[GenericSets[Topic]](1000)
	}

	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30015, kind_30015, parseTopicString, sys.TopicSetsCache)
	return ml
}

func parseTopicString(tag nostr.Tag) (t Topic, ok bool) {
	if t := tag.Value(); t != "" && tag[0] == "t" {
		return Topic(t), true
	}
	return t, false
}
