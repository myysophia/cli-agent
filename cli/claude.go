package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ClaudeCLI 实现 Claude Code CLI
type ClaudeCLI struct{}

// ClaudeOutput 表示 Claude CLI 的 JSON 输出格式
type ClaudeOutput struct {
	Type         string  `json:"type,omitempty"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	DurationMS   int     `json:"duration_ms,omitempty"`
}

func NewClaudeCLI() *ClaudeCLI {
	return &ClaudeCLI{}
}

func (c *ClaudeCLI) Name() string {
	return "claude"
}

func (c *ClaudeCLI) Run(opts *RunOptions) (string, error) {
	var args []string

	// 构建基础参数
	if opts.SessionID != "" {
		args = []string{"-p", opts.Prompt, "--output-format", "json", "--resume", opts.SessionID}
		log.Printf("🔄 [Claude] Resuming session: %s", opts.SessionID)
	} else {
		args = []string{"-p", opts.Prompt, "--output-format", "json"}
		log.Printf("🆕 [Claude] Creating new session")
	}

	// 添加 allowedTools 参数
	if len(opts.AllowedTools) > 0 {
		toolsStr := strings.Join(opts.AllowedTools, ",")
		args = append(args, "--allowedTools", toolsStr)
		log.Printf("🔧 [Claude] Allowed tools: %s", toolsStr)
	}

	// 添加 permission-mode 参数
	if opts.PermissionMode != "" {
		args = append(args, "--permission-mode", opts.PermissionMode)
		log.Printf("🔐 [Claude] Permission mode: %s", opts.PermissionMode)
	}

	// 添加系统提示词
	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
		log.Printf("🎯 [Claude] System prompt: %s", truncate(opts.SystemPrompt, 50))
	}

	// 添加 Skills
	for _, skill := range opts.Skills {
		args = append(args, "--add-dir", skill)
	}
	if len(opts.Skills) > 0 {
		log.Printf("📚 [Claude] Using %d skill(s): %v", len(opts.Skills), opts.Skills)
	}

	log.Printf("⚙️  [Claude] Executing: claude %s", strings.Join(args, " "))

	// 执行命令
	cmd := exec.Command("claude", args...)
	cmd.Env = buildEnv(opts.Env)

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [Claude] Output length: %d bytes", len(output))

	if err != nil {
		log.Printf("❌ [Claude] Execution error: %v", err)
		return "", fmt.Errorf("claude CLI execution failed: %v, output: %s", err, string(output))
	}

	return c.parseOutput(string(output), opts.Prompt)
}

func (c *ClaudeCLI) parseOutput(output string, prompt string) (string, error) {
	// 找到 JSON 起始位置（可能有警告信息在前面）
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		log.Printf("❌ [Claude] No JSON found in output")
		return "", fmt.Errorf("no JSON found in claude output: %s", output)
	}

	if jsonStart > 0 {
		warning := strings.TrimSpace(output[:jsonStart])
		log.Printf("⚠️  [Claude] Warning: %s", truncate(warning, 200))
	}

	// 解析 JSON
	jsonOutput := output[jsonStart:]
	var claudeOut ClaudeOutput
	if err := json.Unmarshal([]byte(jsonOutput), &claudeOut); err != nil {
		log.Printf("❌ [Claude] JSON parse error: %v", err)
		return "", fmt.Errorf("failed to parse claude output: %v", err)
	}

	log.Printf("✨ [Claude] Result preview: %s", truncate(claudeOut.Result, 100))

	// 构建统一输出格式
	result := CLIOutput{
		SessionID: claudeOut.SessionID,
		User:      prompt,
		Response:  claudeOut.Result,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return claudeOut.Result, nil
	}

	return string(jsonBytes), nil
}
