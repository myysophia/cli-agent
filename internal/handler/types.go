package handler

import (
	"encoding/json"
	"strconv"
)

// FlexBool 是一个可以接受布尔值或字符串的类型
type FlexBool bool

// UnmarshalJSON 实现自定义的 JSON 解析
func (fb *FlexBool) UnmarshalJSON(data []byte) error {
	// 尝试解析为布尔值
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexBool(b)
		return nil
	}
	
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		b, err := strconv.ParseBool(s)
		if err != nil {
			*fb = false
			return nil
		}
		*fb = FlexBool(b)
		return nil
	}
	
	*fb = false
	return nil
}

// FlexStringArray 是一个可以接受字符串数组或单个字符串的类型
type FlexStringArray []string

// UnmarshalJSON 实现自定义的 JSON 解析
func (fsa *FlexStringArray) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串数组
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*fsa = FlexStringArray(arr)
		return nil
	}
	
	// 尝试解析为单个字符串
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*fsa = FlexStringArray([]string{})
		} else {
			*fsa = FlexStringArray([]string{s})
		}
		return nil
	}
	
	*fsa = FlexStringArray([]string{})
	return nil
}

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
	Prompt         string          `json:"prompt"`                     // 用户消息（优先使用）
	Message        string          `json:"message"`                    // 用户消息（兼容字段）
	System         string          `json:"system"`
	Profile        string          `json:"profile,omitempty"`          // 可选：指定使用的配置 profile
	CLI            string          `json:"cli,omitempty"`              // 可选：CLI 工具名称（"claude" 或 "codex"，默认 "claude"）
	SessionID      string          `json:"session_id,omitempty"`       // 可选：会话 ID，用于继续之前的对话
	NewSession     FlexBool        `json:"new_session,omitempty"`      // 可选：是否创建新会话（支持布尔值或字符串）
	WorkflowRunID  string          `json:"workflow_run_id,omitempty"`  // 可选：Dify 工作流运行 ID，用于自动管理会话
	AllowedTools   FlexStringArray `json:"allowed_tools,omitempty"`    // 可选：允许使用的 MCP 工具列表（支持数组或字符串）
	PermissionMode string          `json:"permission_mode,omitempty"`  // 可选：权限模式（仅 Claude CLI 支持，如 "bypassPermissions"）
}

// InvokeResponse 表示返回给 Dify 的响应
type InvokeResponse struct {
	Answer string `json:"answer"`
}

// CLIOutput 表示统一的 CLI 输出格式（兼容旧格式）
type CLIOutput struct {
	SessionID string `json:"session_id"`
	User      string `json:"user"`
	Codex     string `json:"codex"`    // 保持兼容性
	Response  string `json:"response"` // 新字段
}
