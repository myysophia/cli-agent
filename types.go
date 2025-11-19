package main

// Message 表示单条对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// InvokeRequest 表示 Dify 发送的请求
type InvokeRequest struct {
	System   string    `json:"system"`
	Messages []Message `json:"messages"`
	Profile  string    `json:"profile,omitempty"` // 可选：指定使用的配置 profile
	CLI      string    `json:"cli,omitempty"`     // 可选：CLI 工具名称（"claude" 或 "codex"，默认 "claude"）
}

// ChatRequest 表示简化的聊天请求
type ChatRequest struct {
	Prompt        string `json:"prompt"`
	System        string `json:"system"`
	Profile       string `json:"profile,omitempty"`          // 可选：指定使用的配置 profile
	CLI           string `json:"cli,omitempty"`              // 可选：CLI 工具名称（"claude" 或 "codex"，默认 "claude"）
	SessionID     string `json:"session_id,omitempty"`       // 可选：会话 ID，用于继续之前的对话
	NewSession    bool   `json:"new_session,omitempty"`      // 可选：是否创建新会话（默认 false，使用 resume --last）
	WorkflowRunID string `json:"workflow_run_id,omitempty"`  // 可选：Dify 工作流运行 ID，用于自动管理会话
}

// InvokeResponse 表示返回给 Dify 的响应
type InvokeResponse struct {
	Answer string `json:"answer"`
}

// ClaudeOutput 表示 Claude CLI 的 JSON 输出格式
type ClaudeOutput struct {
	Type         string  `json:"type,omitempty"`
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	DurationMS   int     `json:"duration_ms,omitempty"`
}

// CodexOutput 表示 Codex CLI 的结构化输出格式
type CodexOutput struct {
	SessionID string `json:"session_id"`
	User      string `json:"user"`
	Codex     string `json:"codex"`
}
