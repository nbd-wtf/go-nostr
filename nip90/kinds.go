package nip90

type Job struct {
	InputKind   int
	OutputKind  int
	Name        string
	Description string
	InputType   string
	Params      []string
}

var Job5000 = Job{
	InputKind:   5000,
	OutputKind:  6000,
	Name:        "Text extraction",
	Description: "Job request to extract text from some kind of input.",
	InputType:   "url",
	Params: []string{
		"alignment",
		"range",
		"raw",
		"segment",
		"word",
	},
}

var Job5001 = Job{
	InputKind:   5001,
	OutputKind:  6001,
	Name:        "Summarization",
	Description: "Summarize input(s)",
	InputType:   "event",
	Params: []string{
		"length",
		"paragraphs",
		"words",
	},
}

var Job5002 = Job{
	InputKind:   5002,
	OutputKind:  6002,
	Name:        "Translation",
	Description: "Translate input(s)",
	InputType:   "event",
	Params:      []string{},
}

var Job5050 = Job{
	InputKind:   5050,
	OutputKind:  6050,
	Name:        "Text Generation",
	Description: "Job request to generate text using AI models.",
	InputType:   "prompt",
	Params: []string{
		"frequency_penalty",
		"max_tokens",
		"model",
		"temperature",
		"top_k",
		"top_p",
	},
}

var Job5100 = Job{
	InputKind:   5100,
	OutputKind:  6100,
	Name:        "Image Generation",
	Description: "Job request to generate Images using AI models.",
	InputType:   "text",
	Params: []string{
		"${width}x${height}",
		"1024x768",
		"512x512",
		"lora",
		"model",
		"negative_prompt",
		"ratio",
		"size",
	},
}

var Job5200 = Job{
	InputKind:   5200,
	OutputKind:  6200,
	Name:        "Video Conversion",
	Description: "Job request to convert a Video to another Format.",
	InputType:   "url",
	Params:      []string{},
}

var Job5201 = Job{
	InputKind:   5201,
	OutputKind:  6201,
	Name:        "Video Translation",
	Description: "Job request to translate video audio content into target language with or without subtitles.",
	InputType:   "url",
	Params: []string{
		"format",
		"language",
		"range",
		"subtitle",
	},
}

var Job5202 = Job{
	InputKind:   5202,
	OutputKind:  6202,
	Name:        "Image-to-Video Conversion",
	Description: "Job request to convert a static Image to a a short animated video clip",
	InputType:   "url",
	Params:      []string{},
}

var Job5250 = Job{
	InputKind:   5250,
	OutputKind:  6250,
	Name:        "Text-to-Speech Generation",
	Description: "Job request to convert text input to an audio file.",
	InputType:   "text",
	Params:      []string{},
}

var Job5300 = Job{
	InputKind:   5300,
	OutputKind:  6300,
	Name:        "Nostr Content Discovery",
	Description: "Job request to discover nostr-native content",
	InputType:   "text",
	Params:      []string{},
}

var Job5301 = Job{
	InputKind:   5301,
	OutputKind:  6301,
	Name:        "Nostr People Discovery",
	Description: "Job request to discover nostr pubkeys",
	InputType:   "text",
	Params:      []string{},
}

var Job5302 = Job{
	InputKind:   5302,
	OutputKind:  6302,
	Name:        "Nostr Content Search",
	Description: "Job to search for notes based on a prompt",
	InputType:   "text",
	Params: []string{
		"max_results",
		"since",
		"until",
		"users",
	},
}

var Job5303 = Job{
	InputKind:   5303,
	OutputKind:  6303,
	Name:        "Nostr People Search",
	Description: "Job to search for profiles based on a prompt",
	InputType:   "text",
	Params:      []string{},
}

var Job5400 = Job{
	InputKind:   5400,
	OutputKind:  6400,
	Name:        "Nostr Event Count",
	Description: "Job request to count matching events",
	InputType:   "text",
	Params: []string{
		"content",
		"group",
		"pubkey",
		"relay",
		"reply",
		"root",
	},
}

var Job5500 = Job{
	InputKind:   5500,
	OutputKind:  6500,
	Name:        "Malware Scanning",
	Description: "Job request to perform a Malware Scan on files.",
	InputType:   "",
	Params:      []string{},
}

var Job5900 = Job{
	InputKind:   5900,
	OutputKind:  6900,
	Name:        "Nostr Event Time Stamping",
	Description: "NIP-03 Timestamping of nostr events",
	InputType:   "event",
	Params:      []string{},
}

var Job5901 = Job{
	InputKind:   5901,
	OutputKind:  6901,
	Name:        "OP_RETURN Creation",
	Description: "Create a bitcoin transaction with an OP_RETURN",
	InputType:   "text",
	Params:      []string{},
}

var Job5905 = Job{
	InputKind:   5905,
	OutputKind:  6905,
	Name:        "Nostr Event Publish Schedule",
	Description: "Schedule nostr events for future publishing",
	InputType:   "text",
	Params:      []string{},
}

var Job5970 = Job{
	InputKind:   5970,
	OutputKind:  6970,
	Name:        "Event PoW Delegation",
	Description: "Delegate PoW of an event to a provider.",
	InputType:   "text",
	Params:      []string{},
}

var Jobs = []Job{
	Job5000,
	Job5001,
	Job5002,
	Job5050,
	Job5100,
	Job5200,
	Job5201,
	Job5202,
	Job5250,
	Job5300,
	Job5301,
	Job5302,
	Job5303,
	Job5400,
	Job5500,
	Job5900,
	Job5901,
	Job5905,
	Job5970,
}
