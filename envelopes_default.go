//go:build !sonic

package nostr

import (
	"errors"
	"strings"
)

func NewMessageParser() MessageParser {
	return messageParser{}
}

type messageParser struct{}

func (messageParser) ParseMessage(message string) (Envelope, error) {
	firstQuote := strings.IndexRune(message, '"')
	if firstQuote == -1 {
		return nil, errors.New("malformed json")
	}
	secondQuote := strings.IndexRune(message[firstQuote+1:], '"')
	if secondQuote == -1 {
		return nil, errors.New("malformed json")
	}
	label := message[firstQuote+1 : firstQuote+1+secondQuote]

	var v Envelope
	switch label {
	case "EVENT":
		v = &EventEnvelope{}
	case "REQ":
		v = &ReqEnvelope{}
	case "COUNT":
		v = &CountEnvelope{}
	case "NOTICE":
		x := NoticeEnvelope("")
		v = &x
	case "EOSE":
		x := EOSEEnvelope("")
		v = &x
	case "OK":
		v = &OKEnvelope{}
	case "AUTH":
		v = &AuthEnvelope{}
	case "CLOSED":
		v = &ClosedEnvelope{}
	case "CLOSE":
		x := CloseEnvelope("")
		v = &x
	default:
		return nil, UnknownLabel
	}

	if err := v.FromJSON(message); err != nil {
		return nil, err
	}
	return v, nil
}
