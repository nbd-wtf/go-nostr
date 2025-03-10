//go:build !sonic

package nostr

import (
	"bytes"
	"errors"
)

func NewMessageParser() MessageParser {
	return messageParser{}
}

type messageParser struct{}

func (messageParser) ParseMessage(message []byte) (Envelope, error) {
	firstComma := bytes.Index(message, []byte{','})
	if firstComma == -1 {
		return nil, errors.New("malformed json")
	}
	label := message[0:firstComma]

	var v Envelope
	switch {
	case bytes.Contains(label, labelEvent):
		v = &EventEnvelope{}
	case bytes.Contains(label, labelReq):
		v = &ReqEnvelope{}
	case bytes.Contains(label, labelCount):
		v = &CountEnvelope{}
	case bytes.Contains(label, labelNotice):
		x := NoticeEnvelope("")
		v = &x
	case bytes.Contains(label, labelEose):
		x := EOSEEnvelope("")
		v = &x
	case bytes.Contains(label, labelOk):
		v = &OKEnvelope{}
	case bytes.Contains(label, labelAuth):
		v = &AuthEnvelope{}
	case bytes.Contains(label, labelClosed):
		v = &ClosedEnvelope{}
	case bytes.Contains(label, labelClose):
		x := CloseEnvelope("")
		v = &x
	default:
		return nil, UnknownLabel
	}

	if err := v.UnmarshalJSON(message); err != nil {
		return nil, err
	}
	return v, nil
}
