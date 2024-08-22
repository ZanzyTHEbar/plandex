package main

type ContextVars struct {
	Changes        string `json:"changes"`
	FilePath       string `json:"filePath"`
	PostBuildState string `json:"postBuildState"`
	PreBuildState  string `json:"preBuildState"`
}

type Context struct {
	Prompt string      `json:"prompt"`
	Vars   ContextVars `json:"vars"`
}

type Function struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

type Output struct {
	Function Function `json:"function"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
}

type RequestPayload struct {
	Context Context  `json:"context"`
	Output  []Output `json:"output"`
}

// ResponsePayload represents the response that the webhook will send back
type ResponsePayload struct {
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score,omitempty"`  // Score is optional
	Reason string  `json:"reason,omitempty"` // Reason is optional
}
