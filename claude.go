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

// runCLI 执行指定的 CLI 工具并返回结果
// 参数：
//   - cliName: CLI 工具名称（"claude" 或 "codex"，为空则默认 "claude"）
//   - prompt: 拼接好的对话内容
//   - systemPrompt: 系统提示词（可为空）
//   - profileName: 配置 profile 名称（可为空，使用默认）
// 返回：
//   - result: CLI 的回答
//   - error: 执行错误
func runCLI(cliName string, prompt string, systemPrompt string, profileName string) (string, error) {
	var cliSource string
	
	// 确定使用的 CLI 工具
	if cliName != "" {
		// 请求中指定了 CLI
		cliSource = "request"
	} else if globalConfig != nil {
		// 尝试从 profile 获取 CLI 配置
		profile, err := globalConfig.getProfile(profileName)
		if err == nil && profile.CLI != "" {
			cliName = profile.CLI
			cliSource = "profile"
		}
	}
	
	// 如果还是空，使用默认值
	if cliName == "" {
		cliName = "claude"
		cliSource = "default"
	}
	
	log.Printf("🔧 CLI tool: %s (from %s)", cliName, cliSource)
	
	// 根据不同的 CLI 工具构建命令参数
	var args []string
	var fullPrompt string
	
	if cliName == "codex" {
		// Codex CLI 使用 exec 子命令，添加 sandbox 参数以支持联网
		args = []string{"exec", "--model", "gpt-5.1", "--sandbox", "danger-full-access"}
		
		// Codex 需要将 system prompt 和 prompt 合并
		if systemPrompt != "" {
			fullPrompt = fmt.Sprintf("System: %s\n\n%s", systemPrompt, prompt)
			log.Printf("🎯 Using system prompt: %s", truncate(systemPrompt, 50))
		} else {
			fullPrompt = prompt
		}
		
		// Codex 的 prompt 作为最后一个参数
		args = append(args, fullPrompt)
	} else {
		// Claude CLI 使用 --print 参数
		args = []string{"--print", prompt, "--output-format", "json", "--allowedTools", "WebSearch"}
		
		// 如果 systemPrompt 非空，追加参数
		if systemPrompt != "" {
			args = append(args, "--append-system-prompt", systemPrompt)
			log.Printf("🎯 Using system prompt: %s", truncate(systemPrompt, 50))
		}
		
		// 添加 Skills 支持（使用 --add-dir 参数）
		if globalConfig != nil {
			profile, err := globalConfig.getProfile(profileName)
			if err == nil && len(profile.Skills) > 0 {
				for _, skill := range profile.Skills {
					args = append(args, "--add-dir", skill)
				}
				log.Printf("📚 Using %d skill(s): %v", len(profile.Skills), profile.Skills)
			}
		}
	}
	
	log.Printf("⚙️  Executing: %s %s", cliName, strings.Join(args, " "))
	
	// 执行命令
	cmd := exec.Command(cliName, args...)
	
	// 如果有配置文件，应用环境变量
	if globalConfig != nil {
		profile, err := globalConfig.getProfile(profileName)
		if err != nil {
			log.Printf("⚠️  %v, using default environment", err)
		} else {
			// log.Printf("🔧 Using profile: %s (%s)", profileName, profile.Name)
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
		return "", fmt.Errorf("%s CLI execution failed: %v, output: %s", cliName, err, string(output))
	}
	
	outputStr := string(output)
	
	// Codex CLI 直接返回文本，不是 JSON
	if cliName == "codex" {
		// log.Printf("✨ Codex result preview: %s", truncate(outputStr, 100))
		return strings.TrimSpace(outputStr), nil
	}
	
	// Claude CLI 返回 JSON 格式
	log.Printf("🔍 Parsing JSON output...")
	
	// Claude CLI 可能会在 JSON 之前输出警告信息，需要找到 JSON 的起始位置
	jsonStart := strings.Index(outputStr, "{")
	if jsonStart == -1 {
		log.Printf("❌ No JSON found in output")
		log.Printf("📄 Raw output: %s", truncate(outputStr, 500))
		return "", fmt.Errorf("no JSON found in %s output: %s", cliName, outputStr)
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
		return "", fmt.Errorf("failed to parse %s output: %v, raw output: %s", cliName, err, jsonOutput)
	}
	
	log.Printf("✨ Result preview: %s", truncate(claudeOut.Result, 100))
	
	// 返回 Result 字段
	return claudeOut.Result, nil
}
