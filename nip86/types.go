package nip86

type Request struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type Response struct {
	Result any    `json:"result"`
	Error  string `json:"error,omitempty"`
}
