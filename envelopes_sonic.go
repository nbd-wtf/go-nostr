package nostr

import (
	"encoding/hex"
	"fmt"

	"github.com/bytedance/sonic/ast"
)

type SonicMessageParser struct{}

func (smp *SonicMessageParser) ParseMessage(message []byte) (Envelope, error) {
	var err error

	tlarr, _ := ast.NewParser(string(message)).Parse()
	label, _ := tlarr.Index(0).StrictString()

	var v Envelope
	switch label {
	case "EVENT":
		env := &EventEnvelope{}
		sndN := tlarr.Index(1)
		var evtN *ast.Node
		switch sndN.TypeSafe() {
		case ast.V_STRING:
			subId, _ := sndN.StrictString()
			env.SubscriptionID = &subId
			evtN = tlarr.Index(2)
		case ast.V_OBJECT:
			evtN = sndN
		}
		err = eventFromSonicAst(&env.Event, evtN)
		v = env
	case "REQ":
		env := &ReqEnvelope{}
		nodes, _ := tlarr.ArrayUseNode()
		env.SubscriptionID, _ = nodes[1].StrictString()
		env.Filters = make(Filters, len(nodes)-2)
		for i, node := range nodes[2:] {
			err = filterFromSonicAst(&env.Filters[i], &node)
		}
		v = env
	case "COUNT":
		env := &CountEnvelope{}
		env.SubscriptionID, _ = tlarr.Index(1).StrictString()
		trdN := tlarr.Index(2)
		if countN := trdN.Get("count"); countN.Exists() {
			count, _ := countN.Int64()
			env.Count = &count
			hll, _ := trdN.Get("hll").StrictString()
			if len(hll) == 512 {
				env.HyperLogLog, _ = hex.DecodeString(hll)
			}
		} else {
			err = filterFromSonicAst(&env.Filter, trdN)
		}
		v = env
	case "NOTICE":
		notice, _ := tlarr.Index(1).StrictString()
		env := NoticeEnvelope(notice)
		v = &env
	case "EOSE":
		subId, _ := tlarr.Index(1).StrictString()
		env := EOSEEnvelope(subId)
		v = &env
	case "OK":
		env := &OKEnvelope{}
		env.EventID, _ = tlarr.Index(1).StrictString()
		env.OK, _ = tlarr.Index(2).Bool()
		env.Reason, _ = tlarr.Index(3).StrictString()
		v = env
	case "AUTH":
		env := &AuthEnvelope{}
		sndN := tlarr.Index(1)
		switch sndN.TypeSafe() {
		case ast.V_STRING:
			challenge, _ := sndN.StrictString()
			env.Challenge = &challenge
		case ast.V_OBJECT:
			err = eventFromSonicAst(&env.Event, sndN)
		}
		v = env
	case "CLOSED":
		env := &ClosedEnvelope{}
		env.SubscriptionID, _ = tlarr.Index(1).StrictString()
		env.Reason, _ = tlarr.Index(2).StrictString()
		v = env
	case "CLOSE":
		reason, _ := tlarr.Index(1).StrictString()
		env := CloseEnvelope(reason)
		v = &env
	default:
		return nil, UnknownLabel
	}

	return v, err
}

func eventFromSonicAst(evt *Event, node *ast.Node) error {
	evt.ID, _ = node.Get("id").StrictString()
	evt.PubKey, _ = node.Get("pubkey").StrictString()
	evt.Content, _ = node.Get("content").StrictString()
	evt.Sig, _ = node.Get("sig").StrictString()
	kind, _ := node.Get("kind").Int64()
	evt.Kind = int(kind)
	createdAt, _ := node.Get("created_at").Int64()
	evt.CreatedAt = Timestamp(createdAt)
	tagsN, err := node.Get("tags").ArrayUseNode()
	if err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}
	evt.Tags = make(Tags, len(tagsN))
	for i, tagN := range tagsN {
		itemsN, err := tagN.ArrayUseNode()
		if err != nil {
			return fmt.Errorf("invalid tag: %w", err)
		}
		tag := make(Tag, len(itemsN))

		for j, itemN := range itemsN {
			tag[j], _ = itemN.StrictString()
		}

		evt.Tags[i] = tag
	}
	return nil
}

func filterFromSonicAst(filter *Filter, node *ast.Node) error {
	var err error

	node.ForEach(func(path ast.Sequence, node *ast.Node) bool {
		switch *path.Key {
		case "limit":
			limit, _ := node.Int64()
			filter.Limit = int(limit)
			filter.LimitZero = filter.Limit == 0
		case "since":
			since, _ := node.Int64()
			filter.Since = (*Timestamp)(&since)
		case "until":
			until, _ := node.Int64()
			filter.Until = (*Timestamp)(&until)
		case "search":
			filter.Search, _ = node.StrictString()
		case "ids":
			idsN, _ := node.ArrayUseNode()
			filter.IDs = make([]string, len(idsN))
			for i, idN := range idsN {
				filter.IDs[i], _ = idN.StrictString()
			}
		case "authors":
			authorsN, _ := node.ArrayUseNode()
			filter.Authors = make([]string, len(authorsN))
			for i, authorN := range authorsN {
				filter.Authors[i], _ = authorN.StrictString()
			}
		case "kinds":
			kindsN, _ := node.ArrayUseNode()
			filter.Kinds = make([]int, len(kindsN))
			for i, kindN := range kindsN {
				kind, _ := kindN.Int64()
				filter.Kinds[i] = int(kind)
			}
		default:
			if len(*path.Key) > 1 && (*path.Key)[0] == '#' {
				if filter.Tags == nil {
					filter.Tags = make(TagMap, 2)
				}
				tagsN, _ := node.ArrayUseNode()
				tags := make([]string, len(tagsN))
				for i, authorN := range tagsN {
					tags[i], _ = authorN.StrictString()
				}
				filter.Tags[(*path.Key)[1:]] = tags
			} else {
				err = fmt.Errorf("unexpected field '%s'", *path.Key)
				return false
			}
		}
		return true
	})

	return err
}
