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

// parseCodexOutput 解析 Codex CLI 的输出，提取关键信息并返回 JSON 格式
// 保留：session id、user 问题、codex 回答（过滤掉工具调用和 thinking 部分）
func parseCodexOutput(output string) string {
	lines := strings.Split(output, "\n")
	var sessionID, userPrompt string
	var lastCodexIndex int = -1
	
	// 第一遍：找到所有关键位置
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 提取 session id
		if strings.HasPrefix(trimmed, "session id:") {
			sessionID = strings.TrimSpace(strings.TrimPrefix(trimmed, "session id:"))
		}
		
		// 检测 user 部分
		if trimmed == "user" {
			// 下一行是用户的问题
			if i+1 < len(lines) {
				userPrompt = strings.TrimSpace(lines[i+1])
			}
		}
		
		// 记录最后一个 "codex" 标记的位置
		if trimmed == "codex" {
			lastCodexIndex = i
		}
	}
	
	// 第二遍：从最后一个 "codex" 标记开始收集答案
	var codexAnswerLines []string
	if lastCodexIndex != -1 {
		for j := lastCodexIndex + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			
			// 遇到 "tokens used" 表示结束
			if strings.HasPrefix(trimmed, "tokens used") {
				break
			}
			
			// 收集非空行
			if trimmed != "" {
				codexAnswerLines = append(codexAnswerLines, trimmed)
			}
		}
	}
	
	codexAnswer := strings.Join(codexAnswerLines, "\n")
	
	// 构建 JSON 格式的输出
	codexOut := CodexOutput{
		SessionID: sessionID,
		User:      userPrompt,
		Codex:     codexAnswer,
	}
	
	jsonBytes, err := json.Marshal(codexOut)
	if err != nil {
		log.Printf("❌ Failed to marshal Codex output to JSON: %v", err)
		// 如果 JSON 序列化失败，返回原始格式
		return fmt.Sprintf("session id: %s\nuser: %s\ncodex: %s", sessionID, userPrompt, codexAnswer)
	}
	
	return string(jsonBytes)
}

// runCLI 执行指定的 CLI 工具并返回结果
// 参数：
//   - cliName: CLI 工具名称（"claude" 或 "codex"，为空则默认 "claude"）
//   - prompt: 拼接好的对话内容
//   - systemPrompt: 系统提示词（可为空）
//   - profileName: 配置 profile 名称（可为空，使用默认）
//   - sessionID: 会话 ID（可为空，用于继续之前的对话）
//   - newSession: 是否创建新会话（true=创建新会话，false=resume last）
// 返回：
//   - result: CLI 的回答
//   - error: 执行错误
func runCLI(cliName string, prompt string, systemPrompt string, profileName string, sessionID string, newSession bool) (string, error) {
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
	
	if cliName == "codex" {
		// 如果提供了 sessionID，使用 resume 命令继续指定会话
		if sessionID != "" {
			args = []string{"exec", "resume", sessionID, prompt}
			log.Printf("🔄 Resuming session: %s", sessionID)
		} else if newSession {
			// 创建新会话
			args = []string{"exec", "--model", "gpt-5.1", "--sandbox", "danger-full-access", prompt}
			log.Printf("🆕 Creating new session")
		} else {
			// 没有 sessionID 且不是新会话，使用 --last 继续最近的会话
			// 注意：--last 不能接受位置参数，prompt 必须通过 stdin 传入
			args = []string{"exec", "resume", "--last"}
			log.Printf("🔄 Resuming last session")
		}
	} else {
		// Claude CLI
		// 如果提供了 sessionID，使用 --resume 继续指定会话
		if sessionID != "" {
			args = []string{"-p", prompt, "--output-format", "json", "--allowedTools", "WebSearch", "--resume", sessionID}
			log.Printf("🔄 Resuming session: %s", sessionID)
		} else {
			// Claude CLI 的 -p 模式不支持自动 resume last
			// 没有 sessionID 时总是创建新会话
			args = []string{"-p", prompt, "--output-format", "json", "--allowedTools", "WebSearch"}
			if newSession {
				log.Printf("🆕 Creating new session")
			} else {
				log.Printf("🆕 Creating new session (Claude -p mode requires explicit session ID for resume)")
			}
		}
		
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
	
	// 如果是 codex resume --last，通过 stdin 传入 prompt
	if cliName == "codex" && sessionID == "" && !newSession && len(args) > 2 && args[2] == "--last" {
		cmd.Stdin = strings.NewReader(prompt)
		log.Printf("📝 Sending prompt via stdin")
	}
	
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
		// 解析 Codex 输出，提取关键信息
		result := parseCodexOutput(outputStr)
		// log.Printf("✨ Codex result preview: %s", truncate(result, 100))
		return result, nil
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
	
	// 构建统一的 JSON 格式输出（与 Codex 保持一致）
	claudeOutput := CodexOutput{
		SessionID: claudeOut.SessionID,
		User:      prompt,
		Codex:     claudeOut.Result, // 使用 Codex 字段名保持一致
	}
	
	jsonBytes, err := json.Marshal(claudeOutput)
	if err != nil {
		log.Printf("❌ Failed to marshal Claude output to JSON: %v", err)
		// 如果 JSON 序列化失败，返回原始格式
		return claudeOut.Result, nil
	}
	
	return string(jsonBytes), nil
}
