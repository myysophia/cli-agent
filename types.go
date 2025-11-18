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
