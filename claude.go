package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// buildPrompt 将 messages 拼接成单个 prompt 字符串
// 格式：
// User: <content>
// Assistant: <content>
// User: <content>
func buildPrompt(messages []Message) string {
	var parts []string
	for _, msg := range messages {
		var prefix string
		if msg.Role == "user" {
			prefix = "User:"
		} else {
			prefix = "Assistant:"
		}
		parts = append(parts, fmt.Sprintf("%s %s", prefix, msg.Content))
	}
	result := strings.Join(parts, "\n")
	log.Printf("🔍 Prompt preview: %s...", truncate(result, 100))
	return result
}

// truncate 截断字符串用于日志显示
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// runClaude 执行 Claude CLI 并返回结果
// 参数：
//   - prompt: 拼接好的对话内容
//   - systemPrompt: 系统提示词（可为空）
//   - profileName: 配置 profile 名称（可为空，使用默认）
// 返回：
//   - result: Claude 的回答
//   - error: 执行错误
func runClaude(prompt string, systemPrompt string, profileName string) (string, error) {
	// 构建命令参数数组
	args := []string{"--print", prompt, "--output-format", "json", "--allowedTools", "WebSearch"}
	
	// 如果 systemPrompt 非空，追加参数
	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
		log.Printf("🎯 Using system prompt: %s", truncate(systemPrompt, 50))
	}
	
	log.Printf("⚙️  Executing: claude %s", strings.Join(args, " "))
	
	// 执行命令
	cmd := exec.Command("claude", args...)
	
	// 如果有配置文件，应用环境变量
	if globalConfig != nil {
		profile, err := globalConfig.getProfile(profileName)
		if err != nil {
			log.Printf("⚠️  %v, using default environment", err)
		} else {
			log.Printf("🔧 Using profile: %s (%s)", profileName, profile.Name)
			// 设置环境变量
			cmd.Env = append(cmd.Env, "PATH="+os.Getenv("PATH"))
			for key, value := range profile.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}
	
	output, err := cmd.CombinedOutput()
	
	log.Printf("📊 CLI output length: %d bytes", len(output))
	
	// 检查命令执行错误
	if err != nil {
		log.Printf("❌ CLI execution error: %v", err)
		log.Printf("📄 Raw output: %s", truncate(string(output), 500))
		return "", fmt.Errorf("claude CLI execution failed: %v, output: %s", err, string(output))
	}
	
	log.Printf("🔍 Parsing JSON output...")
	
	// Claude CLI 可能会在 JSON 之前输出警告信息，需要找到 JSON 的起始位置
	outputStr := string(output)
	jsonStart := strings.Index(outputStr, "{")
	if jsonStart == -1 {
		log.Printf("❌ No JSON found in output")
		log.Printf("📄 Raw output: %s", truncate(outputStr, 500))
		return "", fmt.Errorf("no JSON found in claude output: %s", outputStr)
	}
	
	// 如果有警告信息，记录下来
	if jsonStart > 0 {
		warning := strings.TrimSpace(outputStr[:jsonStart])
		log.Printf("⚠️  CLI warning: %s", truncate(warning, 200))
	}
	
	// 解析 JSON 输出到 ClaudeOutput 结构体
	jsonOutput := outputStr[jsonStart:]
	var claudeOut ClaudeOutput
	if err := json.Unmarshal([]byte(jsonOutput), &claudeOut); err != nil {
		log.Printf("❌ JSON parse error: %v", err)
		log.Printf("📄 Raw JSON: %s", truncate(jsonOutput, 500))
		return "", fmt.Errorf("failed to parse claude output: %v, raw output: %s", err, jsonOutput)
	}
	
	log.Printf("✨ Result preview: %s", truncate(claudeOut.Result, 100))
	
	// 返回 Result 字段
	return claudeOut.Result, nil
}
