package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// QwenCLI 实现 Qwen Code CLI
type QwenCLI struct{}

// QwenOutput 表示 Qwen CLI 的 JSON 输出格式
type QwenOutput struct {
	Response string `json:"response,omitempty"`
	Stats    struct {
		Models map[string]interface{} `json:"models,omitempty"`
	} `json:"stats,omitempty"`
}

func NewQwenCLI() *QwenCLI {
	return &QwenCLI{}
}

func (q *QwenCLI) Name() string {
	return "qwen"
}

func (q *QwenCLI) Run(opts *RunOptions) (string, error) {
	var args []string

	// 基础参数：JSON 输出格式
	args = []string{"--output-format", "json"}

	// 模型选择
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
		log.Printf("🤖 [Qwen] Using model: %s", opts.Model)
	}

	// 权限模式
	if opts.PermissionMode == "bypassPermissions" {
		args = append(args, "--yolo")
		log.Printf("🔐 [Qwen] YOLO mode enabled")
	}

	// 允许的工具
	if len(opts.AllowedTools) > 0 {
		for _, tool := range opts.AllowedTools {
			args = append(args, "--allowed-tools", tool)
		}
		log.Printf("🔧 [Qwen] Allowed tools: %v", opts.AllowedTools)
	}

	// 添加 prompt（作为位置参数）
	args = append(args, opts.Prompt)

	log.Printf("⚙️  [Qwen] Executing: qwen %s", strings.Join(args, " "))

	cmd := exec.Command("qwen", args...)
	cmd.Env = buildEnv(opts.Env)

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [Qwen] Output length: %d bytes", len(output))

	if err != nil {
		log.Printf("❌ [Qwen] Execution error: %v", err)
		return "", fmt.Errorf("qwen CLI execution failed: %v, output: %s", err, string(output))
	}

	return q.parseOutput(string(output), opts.Prompt)
}

func (q *QwenCLI) parseOutput(output string, prompt string) (string, error) {
	// Qwen 输出可能包含前置信息，需要找到 JSON 的起始位置
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		log.Printf("❌ [Qwen] No JSON found in output")
		return "", fmt.Errorf("no JSON found in qwen output: %s", output)
	}

	jsonOutput := output[jsonStart:]

	var qwenOut QwenOutput
	if err := json.Unmarshal([]byte(jsonOutput), &qwenOut); err != nil {
		log.Printf("❌ [Qwen] JSON parse error: %v", err)
		return strings.TrimSpace(output), nil
	}

	response := qwenOut.Response
	if response == "" {
		response = strings.TrimSpace(output)
	}

	log.Printf("✨ [Qwen] Result preview: %s", truncate(response, 100))

	result := CLIOutput{
		SessionID: "",
		User:      prompt,
		Response:  response,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return response, nil
	}

	return string(jsonBytes), nil
}
