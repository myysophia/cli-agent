package cli

import "os"

// truncate 截断字符串用于日志显示
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// buildEnv 构建命令执行的环境变量
func buildEnv(envMap map[string]string) []string {
	env := os.Environ()
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}
	return env
}
