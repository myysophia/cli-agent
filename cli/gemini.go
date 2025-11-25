package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// GeminiCLI 实现 Google Gemini CLI
type GeminiCLI struct{}

// GeminiOutput 表示 Gemini CLI 的 JSON 输出格式
type GeminiOutput struct {
	Response string `json:"response,omitempty"`
	Stats    struct {
		Models map[string]interface{} `json:"models,omitempty"`
	} `json:"stats,omitempty"`
}

func NewGeminiCLI() *GeminiCLI {
	return &GeminiCLI{}
}

func (g *GeminiCLI) Name() string {
	return "gemini"
}

func (g *GeminiCLI) Run(opts *RunOptions) (string, error) {
	var args []string

	// 基础参数：JSON 输出格式
	args = []string{"--output-format", "json"}

	// 会话管理
	if opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
		log.Printf("🔄 [Gemini] Resuming session: %s", opts.SessionID)
	} else if !opts.NewSession {
		// 默认继续最近的会话
		args = append(args, "--resume", "latest")
		log.Printf("🔄 [Gemini] Resuming latest session")
	} else {
		log.Printf("🆕 [Gemini] Creating new session")
	}

	// 模型选择
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
		log.Printf("🤖 [Gemini] Using model: %s", opts.Model)
	}

	// 权限模式
	if opts.PermissionMode == "bypassPermissions" {
		args = append(args, "--yolo")
		log.Printf("🔐 [Gemini] YOLO mode enabled")
	}

	// 允许的工具
	if len(opts.AllowedTools) > 0 {
		for _, tool := range opts.AllowedTools {
			args = append(args, "--allowed-tools", tool)
		}
		log.Printf("🔧 [Gemini] Allowed tools: %v", opts.AllowedTools)
	}

	// 添加 prompt（作为位置参数）
	args = append(args, opts.Prompt)

	log.Printf("⚙️  [Gemini] Executing: gemini %s", strings.Join(args, " "))

	cmd := exec.Command("gemini", args...)
	cmd.Env = buildEnv(opts.Env)

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [Gemini] Output length: %d bytes", len(output))

	if err != nil {
		log.Printf("❌ [Gemini] Execution error: %v", err)
		return "", fmt.Errorf("gemini CLI execution failed: %v, output: %s", err, string(output))
	}

	return g.parseOutput(string(output), opts.Prompt)
}

func (g *GeminiCLI) parseOutput(output string, prompt string) (string, error) {
	// Gemini 输出可能包含前置信息（如 "Loaded cached credentials."）
	// 需要找到 JSON 的起始位置
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		log.Printf("❌ [Gemini] No JSON found in output")
		return "", fmt.Errorf("no JSON found in gemini output: %s", output)
	}

	jsonOutput := output[jsonStart:]

	var geminiOut GeminiOutput
	if err := json.Unmarshal([]byte(jsonOutput), &geminiOut); err != nil {
		log.Printf("❌ [Gemini] JSON parse error: %v", err)
		// 尝试使用原始输出
		return strings.TrimSpace(output), nil
	}

	response := geminiOut.Response
	if response == "" {
		response = strings.TrimSpace(output)
	}

	log.Printf("✨ [Gemini] Result preview: %s", truncate(response, 100))

	result := CLIOutput{
		SessionID: "", // Gemini 不返回 session_id
		User:      prompt,
		Response:  response,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return response, nil
	}

	return string(jsonBytes), nil
}
