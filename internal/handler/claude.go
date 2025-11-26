package handler

import (
	"fmt"
	"log"
	"strings"

	"dify-cli-gateway/internal/cli"
)

// buildPrompt 将 messages 拼接成单个 prompt 字符串
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
func runCLI(cliName string, prompt string, systemPrompt string, profileName string, sessionID string, newSession bool, allowedTools []string, permissionMode string) (string, error) {
	var cliSource string

	// 确定使用的 CLI 工具
	if cliName != "" {
		cliSource = "request"
	} else if globalConfig != nil {
		profile, err := globalConfig.getProfile(profileName)
		if err == nil && profile.CLI != "" {
			cliName = profile.CLI
			cliSource = "profile"
		}
	}

	if cliName == "" {
		cliName = "claude"
		cliSource = "default"
	}

	log.Printf("🔧 CLI tool: %s (from %s)", cliName, cliSource)

	// 创建 CLI 实例
	runner, err := cli.NewCLI(cliName)
	if err != nil {
		return "", fmt.Errorf("failed to create CLI: %v", err)
	}

	// 构建执行选项
	opts := &cli.RunOptions{
		Prompt:         prompt,
		SystemPrompt:   systemPrompt,
		SessionID:      sessionID,
		NewSession:     newSession,
		AllowedTools:   allowedTools,
		PermissionMode: permissionMode,
	}

	// 从配置中获取额外选项
	if globalConfig != nil {
		profile, err := globalConfig.getProfile(profileName)
		if err == nil {
			opts.Skills = profile.Skills
			opts.Env = profile.Env
			opts.Model = profile.Model
		} else {
			log.Printf("⚠️  %v, using default environment", err)
		}
	}

	// 执行 CLI
	return runner.Run(opts)
}
