package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CursorCLI 实现 Cursor Agent CLI
type CursorCLI struct{}

// CursorOutput 表示 Cursor Agent 的 JSON 输出格式
type CursorOutput struct {
	Type       string `json:"type,omitempty"`
	Subtype    string `json:"subtype,omitempty"`
	Result     string `json:"result,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
}

func NewCursorCLI() *CursorCLI {
	return &CursorCLI{}
}

func (c *CursorCLI) Name() string {
	return "cursor-agent"
}

func (c *CursorCLI) Run(opts *RunOptions) (string, error) {
	var args []string

	// 基础参数：使用 print 模式（非交互）、强制模式、浏览器支持、JSON 输出
	// --print 参数确保在非交互环境（如 HTTP 请求、crontab）中正常运行
	args = []string{"--print", "--force", "--browser", "--output-format", "json"}

	// 检测是否为 HTTP 请求（非交互环境）
	// HTTP_REQUEST 标志由 handler 设置，用于区分 HTTP 请求和 CLI 直接调用
	isHTTPRequest := opts.Env != nil && opts.Env["HTTP_REQUEST"] == "true"
	
	// 会话管理
	// 在 HTTP 请求中（非交互环境），避免使用 --resume 触发 raw mode 错误
	// 在交互环境中（CLI 直接调用），支持会话恢复以支持多轮对话
	if opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
		log.Printf("🔄 [Cursor] Resuming session: %s", opts.SessionID)
	} else if opts.NewSession {
		log.Printf("🆕 [Cursor] Creating new session (explicit)")
	} else if !isHTTPRequest {
		// 仅在交互环境中使用 --resume 恢复最后一个会话
		args = append(args, "--resume")
		log.Printf("🔄 [Cursor] Resuming last session (interactive mode)")
	} else {
		log.Printf("🆕 [Cursor] Creating new session (HTTP request mode)")
	}

	// 模型选择
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
		log.Printf("🤖 [Cursor] Using model: %s", opts.Model)
	}

	// 工作目录
	if opts.WorkDir != "" {
		args = append(args, "--workspace", opts.WorkDir)
		log.Printf("📁 [Cursor] Workspace: %s", opts.WorkDir)
	}

	// 自动批准 MCP 服务器
	if len(opts.AllowedTools) > 0 || opts.PermissionMode == "bypassPermissions" {
		args = append(args, "--approve-mcps")
		log.Printf("🔧 [Cursor] Auto-approving MCP servers")
	}

	// 强制允许命令
	if opts.PermissionMode == "bypassPermissions" {
		args = append(args, "--force")
		log.Printf("🔐 [Cursor] Force mode enabled")
	}

	// 添加 prompt
	args = append(args, opts.Prompt)

	log.Printf("⚙️  [Cursor] Executing: cursor-agent %s", strings.Join(args, " "))

	cmd := exec.Command("cursor-agent", args...)
	
	// 构建环境变量，添加禁用 TTY 的配置
	env := opts.Env
	if env == nil {
		env = make(map[string]string)
	}
	// 禁用 Ink 的 raw mode，避免在非交互环境中出错
	env["CI"] = "true"
	env["TERM"] = "dumb"
	
	cmd.Env = buildEnv(env)

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [Cursor] Output length: %d bytes", len(output))

	if err != nil {
		log.Printf("❌ [Cursor] Execution error: %v", err)
		return "", fmt.Errorf("cursor-agent CLI execution failed: %v, output: %s", err, string(output))
	}

	return c.parseOutput(string(output), opts.Prompt)
}

func (c *CursorCLI) parseOutput(output string, prompt string) (string, error) {
	// Cursor Agent 输出单行 JSON（type=result 时包含最终结果）
	lines := strings.Split(output, "\n")

	var lastResult string
	var sessionID string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var cursorOut CursorOutput
		if err := json.Unmarshal([]byte(line), &cursorOut); err != nil {
			continue
		}

		// 收集 session ID
		if cursorOut.SessionID != "" {
			sessionID = cursorOut.SessionID
		}

		// 收集结果（type=result 时包含最终答案）
		if cursorOut.Type == "result" && cursorOut.Result != "" {
			lastResult = cursorOut.Result
		}
	}

	// 如果没有解析到结果，使用原始输出
	if lastResult == "" {
		lastResult = strings.TrimSpace(output)
	}

	log.Printf("✨ [Cursor] Result preview: %s", truncate(lastResult, 100))

	result := CLIOutput{
		SessionID: sessionID,
		User:      prompt,
		Response:  lastResult,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return lastResult, nil
	}

	return string(jsonBytes), nil
}
