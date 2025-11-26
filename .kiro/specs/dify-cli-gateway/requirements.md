# Requirements Document

## Introduction

本项目实现一个极简的 Go HTTP 网关服务，作为 Dify 与 Claude CLI 之间的桥梁。接收 Dify 的对话请求，调用 Claude CLI 无头模式，返回结果。MVP 版本专注核心功能，无状态设计。

## Glossary

- **Gateway Service**: Go HTTP 服务，转发请求到 Claude CLI
- **Dify**: 上游大模型编排平台
- **Claude CLI**: Anthropic 命令行工具，支持 --print 无头模式
- **System Prompt**: 系统提示词
- **Messages**: 对话历史消息列表

## Requirements

### Requirement 1

**User Story:** 作为 Dify 平台，我希望通过 HTTP 接口调用 Claude CLI，以便获取模型响应

#### Acceptance Criteria

1. THE Gateway Service SHALL expose a POST endpoint at path "/invoke"
2. WHEN a request is received, THE Gateway Service SHALL accept Content-Type "application/json"
3. THE Gateway Service SHALL parse request body containing system prompt and messages array
4. THE Gateway Service SHALL return JSON response with the model answer
5. THE Gateway Service SHALL listen on port 8080

### Requirement 2

**User Story:** 作为网关服务，我需要将 Dify 的对话数据转换为 Claude CLI 的 prompt 格式

#### Acceptance Criteria

1. THE Gateway Service SHALL concatenate all messages into a single prompt string
2. THE Gateway Service SHALL format each message with role prefix ("User:" or "Assistant:")
3. THE Gateway Service SHALL preserve message order in the prompt
4. WHEN system prompt is provided, THE Gateway Service SHALL use "--append-system-prompt" flag
5. THE Gateway Service SHALL pass the concatenated prompt to Claude CLI

### Requirement 3

**User Story:** 作为网关服务，我需要调用 Claude CLI 无头模式并获取 JSON 输出

#### Acceptance Criteria

1. THE Gateway Service SHALL execute command "claude --print <prompt> --output-format json --allowedTools WebSearch"
2. THE Gateway Service SHALL capture stdout from Claude CLI
3. THE Gateway Service SHALL parse JSON response and extract the "result" field
4. IF CLI exits with non-zero code, THEN THE Gateway Service SHALL return HTTP 500 with error details
5. WHEN CLI succeeds, THE Gateway Service SHALL return HTTP 200 with the result

### Requirement 4

**User Story:** 作为开发者，我希望代码结构清晰简单，便于快速实现和后续扩展

#### Acceptance Criteria

1. THE Gateway Service SHALL define struct Message with role and content fields
2. THE Gateway Service SHALL define struct InvokeRequest with system and messages fields
3. THE Gateway Service SHALL define struct InvokeResponse with answer field
4. THE Gateway Service SHALL implement buildPrompt function to concatenate messages
5. THE Gateway Service SHALL implement runClaude function to execute CLI and parse output

### Requirement 5

**User Story:** 作为 Dify 用户，我希望使用简化的请求格式快速调用 Claude，无需构造复杂的 messages 数组

#### Acceptance Criteria

1. THE Gateway Service SHALL expose a POST endpoint at path "/chat"
2. THE Gateway Service SHALL accept a simplified request with only "prompt" field
3. THE Gateway Service SHALL accept optional "system" field in the simplified request
4. THE Gateway Service SHALL accept optional "profile" field to specify which model configuration to use
5. WHEN simplified request is received, THE Gateway Service SHALL convert it to Claude CLI format
6. THE Gateway Service SHALL return the same response format as "/invoke" endpoint

### Requirement 6

**User Story:** 作为开发者，我希望网关支持多种 CLI 客户端（Claude 和 Codex），以便灵活切换不同的模型提供商

#### Acceptance Criteria

1. THE Gateway Service SHALL accept optional "cli" field to specify which CLI tool to use
2. WHEN "cli" field is "codex", THE Gateway Service SHALL execute codex command
3. WHEN "cli" field is "claude" or empty, THE Gateway Service SHALL execute claude command
4. THE Gateway Service SHALL use the same command arguments format for both CLI tools
5. THE Gateway Service SHALL parse JSON output from both CLI tools in the same way

### Requirement 7

**User Story:** 作为开发者，我希望在配置文件中为每个 profile 指定默认的 CLI 工具，避免每次请求都要传递 CLI 参数

#### Acceptance Criteria

1. THE Gateway Service SHALL support optional "cli" field in profile configuration
2. WHEN profile has "cli" field, THE Gateway Service SHALL use it as default CLI tool
3. WHEN request has "cli" field, THE Gateway Service SHALL override profile's CLI setting
4. WHEN neither profile nor request has "cli" field, THE Gateway Service SHALL use "claude" as default
5. THE Gateway Service SHALL log which CLI tool is being used for each request
