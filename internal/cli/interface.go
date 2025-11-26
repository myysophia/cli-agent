package cli

// CLIRunner 定义 CLI 工具的通用接口
type CLIRunner interface {
	// Name 返回 CLI 工具名称
	Name() string

	// Run 执行 CLI 命令
	// 参数：
	//   - opts: 执行选项
	// 返回：
	//   - result: 执行结果（JSON 格式）
	//   - error: 执行错误
	Run(opts *RunOptions) (string, error)
}

// RunOptions 定义 CLI 执行的通用选项
type RunOptions struct {
	Prompt         string            // 用户输入
	SystemPrompt   string            // 系统提示词
	SessionID      string            // 会话 ID（用于继续对话）
	NewSession     bool              // 是否创建新会话
	AllowedTools   []string          // 允许使用的工具列表
	PermissionMode string            // 权限模式
	Skills         []string          // Skills 路径列表
	Env            map[string]string // 环境变量
	Model          string            // 模型名称
	WorkDir        string            // 工作目录
}

// CLIOutput 定义统一的输出格式
type CLIOutput struct {
	SessionID string `json:"session_id"`
	User      string `json:"user"`
	Response  string `json:"response"`
}
