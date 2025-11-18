# Claude CLI Gateway

一个极简的 Go HTTP 网关服务，将 HTTP 请求桥接到 Claude CLI。通过 HTTP 接口调用 Claude CLI 的无头模式，让任何支持 HTTP 的应用都能使用 Claude CLI 的能力。

## 功能特性

- 提供 HTTP POST 接口 `/invoke` 接收对话请求
- 自动将对话历史转换为 Claude CLI 的 prompt 格式
- 支持系统提示词（system prompt）
- 调用 Claude CLI 无头模式（`--print`）获取响应
- 返回 JSON 格式的结果
- 默认启用 WebSearch 工具

## 使用场景

- **Dify 集成**: 作为自定义模型提供商接入 Dify
- **API 服务**: 为不支持 CLI 的应用提供 Claude 访问能力
- **自动化工具**: 在 CI/CD 或自动化脚本中通过 HTTP 调用 Claude
- **本地开发**: 快速搭建本地 Claude API 服务进行测试

## 前置要求

1. **Go 环境**: Go 1.16 或更高版本
2. **Claude CLI**: 已安装并配置好 Anthropic Claude CLI
   - 确保 `claude` 命令在 PATH 中可用
   - 已完成 Claude CLI 的认证配置

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

调用 Claude CLI 获取模型响应。

**请求格式**:

```json
{
  "profile": "配置名称（可选，默认使用 configs.json 中的 default）",
  "system": "系统提示词（可选）",
  "messages": [
    {"role": "user", "content": "用户消息"},
    {"role": "assistant", "content": "助手回复"},
    {"role": "user", "content": "用户继续提问"}
  ]
}
```

**字段说明**:
- `profile` (string, 可选): 指定使用的配置 profile（如 "minimax", "glm", "kimi"）
- `system` (string, 可选): 系统提示词，用于设定 AI 的行为和角色
- `messages` (array, 必需): 对话历史消息列表
  - `role` (string): 消息角色，可选值 `"user"` 或 `"assistant"`
  - `content` (string): 消息内容

**成功响应** (200 OK):

```json
{
  "answer": "Claude 生成的回答内容"
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
claude-cli-gateway/
├── main.go          # 程序入口，启动 HTTP 服务器
├── handler.go       # HTTP handler 实现
├── claude.go        # Claude CLI 调用逻辑
├── types.go         # 数据结构定义
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
- `cli`: 使用的 CLI 工具（可选，"claude" 或 "codex"，默认 "claude"）
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

#### Codex CLI 配置示例

如果你想使用原生的 OpenAI Codex CLI（GPT-4.1），只需在 profile 中添加 `"cli": "codex"` 字段。由于 Codex CLI 已在本地配置好，不需要设置额外的环境变量：

```json
{
  "profiles": {
    "codex": {
      "name": "OpenAI Codex (GPT-4.1)",
      "cli": "codex",
      "env": {}
    }
  },
  "default": "codex"
}
```

**使用 Codex CLI**：
```bash
# 使用 Codex profile
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "写一个 Python 快速排序",
    "profile": "codex"
  }'

# 或者在请求中临时指定使用 codex（覆盖 profile 配置）
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "写一个 Python 快速排序",
    "profile": "qwen",
    "cli": "codex"
  }'
```

**注意**：
- Codex CLI 需要预先在本地配置好（通过 `codex login` 等命令）
- 网关会调用 `codex exec --model gpt-5.1 --sandbox danger-full-access` 命令
- `--sandbox danger-full-access` 参数允许 Codex 联网访问
- 使用你本地配置的 Codex 认证信息
- 不需要在 `env` 中配置任何 API 密钥或端点
- Codex 返回纯文本格式（不是 JSON）

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
2. **Prompt Builder** (`claude.go`): 将消息数组拼接成 Claude CLI 可用的 prompt
3. **CLI Executor** (`claude.go`): 执行 claude 命令，解析 JSON 输出

### 添加新功能

项目采用模块化设计，便于扩展：

- 添加鉴权：在 `handler.go` 中添加 token 验证逻辑
- 添加日志：引入日志库记录请求详情
- 配置化：使用环境变量或配置文件替代硬编码
- 支持其他 CLI：在 `claude.go` 中抽象 CLI 接口

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

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
