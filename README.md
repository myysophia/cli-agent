# CLI Gateway

一个极简的 Go HTTP 网关服务，将 HTTP 请求桥接到多种 AI CLI 工具。通过统一的 HTTP 接口调用各种 CLI 的无头模式，让任何支持 HTTP 的应用都能使用这些 CLI 的能力。

## 支持的 CLI 工具

| CLI | 说明 | 模型示例 |
|-----|------|----------|
| `claude` | Anthropic Claude Code CLI | claude-sonnet-4, 支持第三方 API |
| `codex` | OpenAI Codex CLI | gpt-5.1 |
| `cursor` | Cursor Agent CLI | auto, gpt-5, sonnet-4 |
| `gemini` | Google Gemini CLI | gemini-2.5-pro, gemini-2.5-flash |
| `qwen` | 阿里 Qwen Code CLI | qwen3-max |

## 功能特性

- 提供 HTTP POST 接口 `/invoke` 和 `/chat` 接收对话请求
- 自动将对话历史转换为 CLI 的 prompt 格式
- 支持系统提示词（system prompt）
- **支持 5 种 CLI 工具**（Claude、Codex、Cursor、Gemini、Qwen）
- 支持 Claude Skills（访问本地文件和目录）
- **支持 MCP 工具调用**（WebFetch、Playwright 等）
- 支持会话管理（session_id 和 resume）
- 调用 CLI 无头模式获取响应
- 返回 JSON 格式的结果
- 支持多配置 profile 管理
- 自动日志记录（按日期分文件）

## 使用场景

- **Dify 集成**: 作为自定义模型提供商接入 Dify
- **API 服务**: 为不支持 CLI 的应用提供 Claude 访问能力
- **自动化工具**: 在 CI/CD 或自动化脚本中通过 HTTP 调用 Claude
- **本地开发**: 快速搭建本地 Claude API 服务进行测试

## 前置要求

1. **Go 环境**: Go 1.16 或更高版本
2. **CLI 工具**: 至少安装并配置好以下一种 CLI：
   - `claude` - Anthropic Claude Code CLI
   - `codex` - OpenAI Codex CLI
   - `cursor-agent` - Cursor Agent CLI
   - `gemini` - Google Gemini CLI
   - `qwen` - 阿里 Qwen Code CLI

## 快速开始

### 1. 构建项目

```bash
# 初始化 Go module（如果还没有）
go mod init claude-cli-gateway

# 构建可执行文件
go build -o claude-cli-gateway
```

### 2. 启动服务

```bash
# 方式一：直接运行可执行文件
./claude-cli-gateway

# 方式二：使用 go run
go run .

# 方式三：使用启动脚本（推荐）
./start.sh
```

服务将在 `http://localhost:8080` 启动。

### 3. 测试接口

使用 curl 测试：

```bash
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "system": "你是一个有帮助的助手",
    "messages": [
      {"role": "user", "content": "什么是 Go 语言？"}
    ]
  }'
```

预期响应：

```json
{
  "answer": "Go 是 Google 开发的一种静态类型、编译型编程语言..."
}
```

## API 文档

### POST /invoke

调用 Claude CLI 获取模型响应（适用于多轮对话）。

**请求格式**:

```json
{
  "profile": "配置名称（可选，默认使用 configs.json 中的 default）",
  "cli": "CLI 工具名称（可选，'claude' 或 'codex'）",
  "system": "系统提示词（可选）",
  "messages": [
    {"role": "user", "content": "用户消息"},
    {"role": "assistant", "content": "助手回复"},
    {"role": "user", "content": "用户继续提问"}
  ]
}
```

**字段说明**:
- `profile` (string, 可选): 指定使用的配置 profile（如 "cursor", "gemini", "codex"）
- `cli` (string, 可选): CLI 工具名称（"claude", "codex", "cursor", "gemini", "qwen"）
- `system` (string, 可选): 系统提示词，用于设定 AI 的行为和角色
- `messages` (array, 必需): 对话历史消息列表
  - `role` (string): 消息角色，可选值 `"user"` 或 `"assistant"`
  - `content` (string): 消息内容

**成功响应** (200 OK):

```json
{
  "answer": "{\"session_id\":\"xxx\",\"user\":\"问题\",\"codex\":\"回答内容\"}"
}
```

### POST /chat

简化的聊天接口（推荐使用）。

**请求格式**:

```json
{
  "profile": "配置名称（可选）",
  "cli": "CLI 工具名称（可选）",
  "prompt": "你的问题",
  "system": "系统提示词（可选）",
  "session_id": "会话ID（可选，用于继续对话）",
  "new_session": false,
  "allowed_tools": ["WebFetch", "playwright"],
  "permission_mode": "bypassPermissions"
}
```

**字段说明**:
- `profile` (string, 可选): 指定使用的配置 profile
- `cli` (string, 可选): CLI 工具名称（"claude", "codex", "cursor", "gemini", "qwen"）
- `prompt` (string, 必需): 用户问题或指令
- `system` (string, 可选): 系统提示词
- `session_id` (string, 可选): 会话 ID，用于继续之前的对话
- `new_session` (boolean, 可选): 是否创建新会话（默认 false）
- `allowed_tools` (array, 可选): 允许使用的 MCP 工具列表
- `permission_mode` (string, 可选): 权限模式（"bypassPermissions" 自动授权）

**成功响应** (200 OK):

```json
{
  "answer": "{\"session_id\":\"xxx\",\"user\":\"问题\",\"codex\":\"回答内容\"}"
}
```

**错误响应**:

- **400 Bad Request**: JSON 格式错误
  ```json
  {"error": "Invalid JSON request body"}
  ```

- **405 Method Not Allowed**: 使用了非 POST 方法
  ```json
  {"error": "Method not allowed"}
  ```

- **500 Internal Server Error**: Claude CLI 执行失败
  ```json
  {"error": "claude CLI execution failed: ..."}
  ```

## 项目结构

```
cli-gateway/
├── main.go          # 程序入口，启动 HTTP 服务器
├── handler.go       # HTTP handler 实现
├── claude.go        # CLI 调用入口
├── types.go         # 数据结构定义
├── config.go        # 配置管理
├── cli/             # CLI 实现包
│   ├── interface.go # CLI 接口定义
│   ├── factory.go   # CLI 工厂函数
│   ├── claude.go    # Claude CLI 实现
│   ├── codex.go     # Codex CLI 实现
│   ├── cursor.go    # Cursor Agent CLI 实现
│   ├── gemini.go    # Gemini CLI 实现
│   ├── qwen.go      # Qwen CLI 实现
│   └── utils.go     # 工具函数
├── configs.json     # 配置文件
├── go.mod           # Go module 定义
├── README.md        # 项目文档
└── start.sh         # 启动脚本
```

## 配置说明

### 基本配置

- **端口**: 8080
- **Claude CLI 工具**: WebSearch（固定启用）
- **输出格式**: JSON
- **日志**: 自动记录到 `logs/` 目录，按日期分文件（如 `logs/2025-11-18.log`）

### 多配置支持

网关支持多个 Claude API 配置（MiniMax、智谱 GLM、Kimi 等），通过 `configs.json` 文件管理。

#### 配置文件格式

创建 `configs.json` 文件（参考 `configs.example.json`）：

```json
{
  "profiles": {
    "minimax": {
      "name": "MiniMax",
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "your-token",
        "ANTHROPIC_MODEL": "MiniMax-M2"
      }
    },
    "glm": {
      "name": "智谱 GLM",
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "your-token"
      }
    },
    "qwen": {
      "name": "阿里百炼 Qwen",
      "cli": "claude",
      "env": {
        "ANTHROPIC_API_KEY": "your-bailian-api-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max",
        "ANTHROPIC_SMALL_FAST_MODEL": "qwen-flash"
      }
    }
  },
  "default": "minimax"
}
```

**配置字段说明**：
- `name`: Profile 的显示名称
- `cli`: 使用的 CLI 工具（"claude", "codex", "cursor", "gemini", "qwen"）
- `model`: 模型名称（可选，如 "gpt-5.1", "sonnet-4", "gemini-2.5-pro"）
- `skills`: Claude Skills 列表（可选，仅 Claude CLI 支持）
  - 可以是目录路径或文件路径
  - Claude 会读取这些路径下的内容作为上下文
  - 支持多个 skill 路径
- `env`: 环境变量配置
  - `ANTHROPIC_API_KEY` 或 `ANTHROPIC_AUTH_TOKEN`: API 密钥
  - `ANTHROPIC_BASE_URL`: API 端点地址
  - `ANTHROPIC_MODEL`: 默认模型
  - `ANTHROPIC_SMALL_FAST_MODEL`: 快速模型（可选）

#### Claude Skills 配置示例

Claude Skills 允许 Claude 访问本地文件和目录，提升回复质量。例如，让 Claude 读取你的研究报告：

```json
{
  "profiles": {
    "qwen-with-reports": {
      "name": "Qwen with Research Reports",
      "cli": "claude",
      "skills": [
        "./reporter",
        "./docs/research"
      ],
      "env": {
        "ANTHROPIC_API_KEY": "your-api-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

**使用 Skills**：
```bash
# Claude 会自动读取 ./reporter 目录下的文件作为上下文
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "根据研究报告，总结最新的技术趋势",
    "profile": "qwen-with-reports"
  }'
```

**Skills 说明**：
- Skills 路径可以是相对路径或绝对路径
- 支持目录（会递归读取）和单个文件
- Claude 会将这些文件内容作为上下文，提升回复的准确性
- 适合场景：研究报告、文档库、代码库等

#### 原生 CLI 配置示例

以下是各种原生 CLI 工具的配置示例：

```json
{
  "profiles": {
    "codex": {
      "name": "OpenAI Codex (GPT-5.1)",
      "cli": "codex",
      "env": {}
    },
    "cursor": {
      "name": "Cursor Agent",
      "cli": "cursor",
      "model": "auto",
      "env": {}
    },
    "cursor-gpt5": {
      "name": "Cursor Agent (GPT-5)",
      "cli": "cursor",
      "model": "gpt-5",
      "env": {}
    },
    "gemini": {
      "name": "Google Gemini",
      "cli": "gemini",
      "env": {}
    },
    "gemini-pro": {
      "name": "Gemini 2.5 Pro",
      "cli": "gemini",
      "model": "gemini-2.5-pro",
      "env": {}
    },
    "qwen-cli": {
      "name": "Qwen Code CLI",
      "cli": "qwen",
      "env": {}
    }
  },
  "default": "cursor"
}
```

**使用示例**：
```bash
# 使用 Cursor Agent
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "写一个 Python 快速排序", "profile": "cursor"}'

# 使用 Gemini
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "解释量子计算", "profile": "gemini-pro"}'

# 使用 Codex
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "分析这段代码", "profile": "codex"}'

# 使用 Qwen CLI
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "你好", "profile": "qwen-cli"}'
```

**注意**：
- 各 CLI 需要预先在本地配置好认证
- Codex: `codex login`
- Cursor: `cursor-agent login`
- Gemini: 使用 Google 账号认证
- Qwen: 使用阿里云账号认证

#### 使用不同配置

在请求中指定 `profile` 字段：

```bash
# 使用 MiniMax
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "profile": "minimax",
    "system": "你是一个助手",
    "messages": [{"role": "user", "content": "你好"}]
  }'

# 使用智谱 GLM
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "profile": "glm",
    "system": "你是一个助手",
    "messages": [{"role": "user", "content": "你好"}]
  }'

# 不指定 profile，使用默认配置
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "system": "你是一个助手",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### 日志功能

服务启动后会自动：
- 创建 `logs/` 目录
- 按日期生成日志文件（格式：`YYYY-MM-DD.log`）
- 同时输出到控制台和文件
- 记录所有请求、响应和性能指标

查看日志：
```bash
# 查看今天的日志
cat logs/$(date +%Y-%m-%d).log

# 实时监控日志
tail -f logs/$(date +%Y-%m-%d).log

# 查看所有日志文件
ls -lh logs/
```

## 集成示例

### 在 Dify 中使用

1. 在 Dify 中添加自定义模型提供商
2. 配置 API 端点为: `http://localhost:8080/invoke`
3. 设置请求方法为 POST
4. 配置请求格式为上述 JSON 格式

### 在其他应用中使用

任何支持 HTTP 的应用都可以调用此网关：

**Python 示例**:
```python
import requests

response = requests.post('http://localhost:8080/invoke', json={
    "system": "你是一个编程助手",
    "messages": [
        {"role": "user", "content": "如何用 Python 读取文件？"}
    ]
})

print(response.json()['answer'])
```

**JavaScript 示例**:
```javascript
const response = await fetch('http://localhost:8080/invoke', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
        system: "你是一个编程助手",
        messages: [
            {role: "user", content: "如何用 JS 读取文件？"}
        ]
    })
});

const data = await response.json();
console.log(data.answer);
```

## 开发说明

### 核心组件

1. **HTTP Handler** (`handler.go`): 处理 HTTP 请求，解析 JSON，返回响应
2. **CLI 接口** (`cli/interface.go`): 定义统一的 CLI 运行接口
3. **CLI 工厂** (`cli/factory.go`): 根据类型创建对应的 CLI 实例
4. **CLI 实现** (`cli/*.go`): 各 CLI 工具的具体实现

### 添加新 CLI 支持

项目采用接口模式，添加新 CLI 只需：

1. 在 `cli/` 目录创建新文件（如 `newcli.go`）
2. 实现 `CLIRunner` 接口：
   ```go
   type CLIRunner interface {
       Name() string
       Run(opts *RunOptions) (string, error)
   }
   ```
3. 在 `cli/factory.go` 中注册新 CLI

## 故障排查

### 服务无法启动

- 检查端口 8080 是否被占用
- 确认 Go 环境已正确安装

### Claude CLI 调用失败

- 确认 `claude` 命令可在终端中直接运行
- 检查 Claude CLI 是否已完成认证
- 查看错误响应中的详细信息

### JSON 解析错误

- 确认请求的 Content-Type 为 `application/json`
- 检查 JSON 格式是否正确
- 确保 messages 数组不为空

## MCP 工具集成

网关支持调用 MCP (Model Context Protocol) 工具，让 AI 能够访问网页、操作浏览器等。

### 配置 MCP 工具

**Claude CLI MCP 配置** (`~/.claude/settings.json`):
```json
{
  "mcpServers": {
    "fetch": {
      "command": "uvx",
      "args": ["mcp-server-fetch"]
    }
  }
}
```

**Codex CLI MCP 配置** (`~/.codex/config.toml`):
```toml
[mcp]
enabled = true

[mcp_servers.playwright]
command = "npx"
args = ["@playwright/mcp@latest"]
```

### 使用 MCP 工具

**示例 1：使用 Playwright 抓取网页**
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "profile": "codex",
    "prompt": "访问 https://www.hangjianet.com/news?page=1 获取前3条新闻的标题和日期",
    "allowed_tools": ["playwright"],
    "permission_mode": "bypassPermissions"
  }'
```

**示例 2：使用 WebFetch 获取网页内容**
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "profile": "claude-mirror",
    "prompt": "获取 https://example.com 的内容并总结",
    "allowed_tools": ["WebFetch"],
    "permission_mode": "bypassPermissions"
  }'
```

**可用的 MCP 工具**:
- `WebFetch`: 获取网页内容（Claude CLI 内置）
- `WebSearch`: 网络搜索（Claude CLI 内置）
- `playwright`: 浏览器自动化（需要配置 Playwright MCP）
- `fetch`: 网页抓取（需要配置 fetch MCP）

**注意事项**:
- 使用 `allowed_tools` 指定允许的工具列表
- 使用 `permission_mode: "bypassPermissions"` 自动授权工具使用
- Codex CLI 的 Playwright 工具功能更强大，推荐用于网页抓取
- Claude CLI 的 WebFetch 可能有网络限制

## 会话管理

网关支持会话管理，可以继续之前的对话。

**创建新会话**:
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "你好，我是张三",
    "new_session": true
  }'
```

**继续会话**（使用返回的 session_id）:
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "我叫什么名字？",
    "session_id": "xxx-xxx-xxx"
  }'
```

**Dify 工作流集成**（自动管理会话）:
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "你好",
    "workflow_run_id": "dify-workflow-123"
  }'
```

## 相关文档

- [Claude Skills 使用指南](./SKILLS.md) - 详细的 Skills 配置和使用说明
- [配置示例](./configs.example.json) - 各种配置场景的示例
- [更新日志](./CHANGELOG.md) - 版本更新记录

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
