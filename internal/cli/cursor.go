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

	// 基础参数：使用 print 模式、强制模式、浏览器支持、JSON 输出
	args = []string{"-p", "-f", "--browser", "--output-format", "json"}

	// 会话管理
	if opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
		log.Printf("🔄 [Cursor] Resuming session: %s", opts.SessionID)
	} else if opts.NewSession {
		log.Printf("🆕 [Cursor] Creating new session")
	} else {
		// 默认继续最近的会话
		args = append(args, "--resume")
		log.Printf("🔄 [Cursor] Resuming last session")
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
	cmd.Env = buildEnv(opts.Env)

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
