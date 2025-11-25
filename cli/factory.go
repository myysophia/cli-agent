package cli

import "fmt"

// CLIType 定义支持的 CLI 类型
type CLIType string

const (
	CLIClaude CLIType = "claude"
	CLICodex  CLIType = "codex"
	CLICursor CLIType = "cursor"
	CLIGemini CLIType = "gemini"
	CLIQwen   CLIType = "qwen"
)

// NewCLI 根据类型创建对应的 CLI 实例
func NewCLI(cliType string) (CLIRunner, error) {
	switch CLIType(cliType) {
	case CLIClaude, "claude-code":
		return NewClaudeCLI(), nil
	case CLICodex:
		return NewCodexCLI(), nil
	case CLICursor, "cursor-agent":
		return NewCursorCLI(), nil
	case CLIGemini:
		return NewGeminiCLI(), nil
	case CLIQwen, "qwen-code":
		return NewQwenCLI(), nil
	default:
		return nil, fmt.Errorf("unsupported CLI type: %s", cliType)
	}
}

// SupportedCLIs 返回所有支持的 CLI 类型
func SupportedCLIs() []string {
	return []string{
		string(CLIClaude),
		string(CLICodex),
		string(CLICursor),
		string(CLIGemini),
		string(CLIQwen),
	}
}
