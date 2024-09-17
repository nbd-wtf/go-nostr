package nip77

import (
	"bytes"
	"fmt"

	"github.com/mailru/easyjson"
	jwriter "github.com/mailru/easyjson/jwriter"
	"github.com/nbd-wtf/go-nostr"
	"github.com/tidwall/gjson"
)

func ParseNegMessage(message []byte) nostr.Envelope {
	firstComma := bytes.Index(message, []byte{','})
	if firstComma == -1 {
		return nil
	}
	label := message[0:firstComma]

	var v nostr.Envelope
	switch {
	case bytes.Contains(label, []byte("NEG-MSG")):
		v = &MessageEnvelope{}
	case bytes.Contains(label, []byte("NEG-OPEN")):
		v = &OpenEnvelope{}
	case bytes.Contains(label, []byte("NEG-ERR")):
		v = &ErrorEnvelope{}
	case bytes.Contains(label, []byte("NEG-CLOSE")):
		v = &CloseEnvelope{}
	default:
		return nil
	}

	if err := v.UnmarshalJSON(message); err != nil {
		return nil
	}
	return v
}

var (
	_ nostr.Envelope = (*OpenEnvelope)(nil)
	_ nostr.Envelope = (*MessageEnvelope)(nil)
	_ nostr.Envelope = (*CloseEnvelope)(nil)
	_ nostr.Envelope = (*ErrorEnvelope)(nil)
)

type OpenEnvelope struct {
	SubscriptionID string
	Filter         nostr.Filter
	Message        string
}

func (_ OpenEnvelope) Label() string { return "NEG-OPEN" }
func (v OpenEnvelope) String() string {
	b, _ := v.MarshalJSON()
	return string(b)
}

func (v *OpenEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) != 4 {
		return fmt.Errorf("failed to decode NEG-OPEN envelope")
	}

	v.SubscriptionID = arr[1].Str
	v.Message = arr[3].Str
	return easyjson.Unmarshal([]byte(arr[2].Raw), &v.Filter)
}

func (v OpenEnvelope) MarshalJSON() ([]byte, error) {
	res := bytes.NewBuffer(make([]byte, 0, 17+len(v.SubscriptionID)+len(v.Message)+500))

	res.WriteString(`["NEG-OPEN","`)
	res.WriteString(v.SubscriptionID)
	res.WriteString(`",`)

	w := jwriter.Writer{}
	v.Filter.MarshalEasyJSON(&w)
	w.Buffer.DumpTo(res)

	res.WriteString(`,"`)
	res.WriteString(v.Message)
	res.WriteString(`"]`)

	return res.Bytes(), nil
}

type MessageEnvelope struct {
	SubscriptionID string
	Message        string
}

func (_ MessageEnvelope) Label() string { return "NEG-MSG" }
func (v MessageEnvelope) String() string {
	b, _ := v.MarshalJSON()
	return string(b)
}

func (v *MessageEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode NEG-MSG envelope")
	}
	v.SubscriptionID = arr[1].Str
	v.Message = arr[2].Str
	return nil
}

func (v MessageEnvelope) MarshalJSON() ([]byte, error) {
	res := bytes.NewBuffer(make([]byte, 0, 17+len(v.SubscriptionID)+len(v.Message)))

	res.WriteString(`["NEG-MSG","`)
	res.WriteString(v.SubscriptionID)
	res.WriteString(`","`)
	res.WriteString(v.Message)
	res.WriteString(`"]`)

	return res.Bytes(), nil
}

type CloseEnvelope struct {
	SubscriptionID string
}

func (_ CloseEnvelope) Label() string { return "NEG-CLOSE" }
func (v CloseEnvelope) String() string {
	b, _ := v.MarshalJSON()
	return string(b)
}

func (v *CloseEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode NEG-CLOSE envelope")
	}
	v.SubscriptionID = arr[1].Str
	return nil
}

func (v CloseEnvelope) MarshalJSON() ([]byte, error) {
	res := bytes.NewBuffer(make([]byte, 0, 14+len(v.SubscriptionID)))
	res.WriteString(`["NEG-CLOSE","`)
	res.WriteString(v.SubscriptionID)
	res.WriteString(`"]`)
	return res.Bytes(), nil
}

type ErrorEnvelope struct {
	SubscriptionID string
	Reason         string
}

func (_ ErrorEnvelope) Label() string { return "NEG-ERROR" }
func (v ErrorEnvelope) String() string {
	b, _ := v.MarshalJSON()
	return string(b)
}

func (v *ErrorEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode NEG-ERROR envelope")
	}
	v.SubscriptionID = arr[1].Str
	v.Reason = arr[2].Str
	return nil
}

func (v ErrorEnvelope) MarshalJSON() ([]byte, error) {
	res := bytes.NewBuffer(make([]byte, 0, 19+len(v.SubscriptionID)+len(v.Reason)))
	res.WriteString(`["NEG-ERROR","`)
	res.WriteString(v.SubscriptionID)
	res.WriteString(`","`)
	res.WriteString(v.Reason)
	res.WriteString(`"]`)
	return res.Bytes(), nil
}