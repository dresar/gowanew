package bot

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message     string        `json:"message"`
	ChatHistory []ChatMessage `json:"chat_history,omitempty"`
	Model       string        `json:"model,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResult struct {
	Reply     string    `json:"reply"`
	Model     string    `json:"model"`
	LatencyMS int64     `json:"latency_ms"`
	Usage     ChatUsage `json:"usage"`
}

type ToolRequest struct {
	Tool       string         `json:"tool"`
	Parameters map[string]any `json:"parameters"`
}

type ToolResult struct {
	Tool      string `json:"tool"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Output    any    `json:"output"`
}

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
