package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CodexCLI 实现 OpenAI Codex CLI
type CodexCLI struct{}

func NewCodexCLI() *CodexCLI {
	return &CodexCLI{}
}

func (c *CodexCLI) Name() string {
	return "codex"
}

func (c *CodexCLI) Run(opts *RunOptions) (string, error) {
	// Codex CLI 不支持 --allowedTools 和 --permission-mode 参数
	if len(opts.AllowedTools) > 0 {
		log.Printf("⚠️  [Codex] Does not support --allowedTools, using MCP config from ~/.codex/config.toml")
	}
	if opts.PermissionMode != "" {
		log.Printf("⚠️  [Codex] Does not support --permission-mode parameter")
	}

	var args []string
	var useStdin bool

	if opts.SessionID != "" {
		// 继续指定会话
		args = []string{"exec", "resume", opts.SessionID, opts.Prompt}
		log.Printf("🔄 [Codex] Resuming session: %s", opts.SessionID)
	} else if opts.NewSession {
		// 创建新会话
		model := opts.Model
		if model == "" {
			model = "gpt-5.1"
		}
		args = []string{"exec", "--model", model, "--sandbox", "danger-full-access", opts.Prompt}
		log.Printf("🆕 [Codex] Creating new session with model: %s", model)
	} else {
		// 继续最近的会话（通过 stdin 传入 prompt）
		args = []string{"exec", "resume", "--last"}
		useStdin = true
		log.Printf("🔄 [Codex] Resuming last session")
	}

	log.Printf("⚙️  [Codex] Executing: codex %s", strings.Join(args, " "))

	cmd := exec.Command("codex", args...)
	cmd.Env = buildEnv(opts.Env)

	if useStdin {
		cmd.Stdin = strings.NewReader(opts.Prompt)
		log.Printf("📝 [Codex] Sending prompt via stdin")
	}

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [Codex] Output length: %d bytes", len(output))

	if err != nil {
		log.Printf("❌ [Codex] Execution error: %v", err)
		return "", fmt.Errorf("codex CLI execution failed: %v, output: %s", err, string(output))
	}

	return c.parseOutput(string(output))
}

func (c *CodexCLI) parseOutput(output string) (string, error) {
	lines := strings.Split(output, "\n")
	var sessionID, userPrompt string
	var lastCodexIndex int = -1

	// 第一遍：找到所有关键位置
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "session id:") {
			sessionID = strings.TrimSpace(strings.TrimPrefix(trimmed, "session id:"))
		}

		if trimmed == "user" && i+1 < len(lines) {
			userPrompt = strings.TrimSpace(lines[i+1])
		}

		if trimmed == "codex" {
			lastCodexIndex = i
		}
	}

	// 第二遍：从最后一个 "codex" 标记开始收集答案
	var answerLines []string
	if lastCodexIndex != -1 {
		for j := lastCodexIndex + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if strings.HasPrefix(trimmed, "tokens used") {
				break
			}
			if trimmed != "" {
				answerLines = append(answerLines, trimmed)
			}
		}
	}

	answer := strings.Join(answerLines, "\n")

	result := CLIOutput{
		SessionID: sessionID,
		User:      userPrompt,
		Response:  answer,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		log.Printf("❌ [Codex] Failed to marshal output: %v", err)
		return fmt.Sprintf("session id: %s\nuser: %s\ncodex: %s", sessionID, userPrompt, answer), nil
	}

	return string(jsonBytes), nil
}
