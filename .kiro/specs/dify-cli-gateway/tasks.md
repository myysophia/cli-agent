# Implementation Plan

- [x] 1. 初始化 Go 项目结构
  - 创建项目目录
  - 初始化 go.mod 文件，设置 module 名称为 dify-cli-gateway
  - _Requirements: 4.1, 4.2, 4.3_

- [x] 2. 实现数据结构定义
  - [x] 2.1 创建 types.go 文件并定义 Message 结构体
    - 定义 Role 字段（string 类型，JSON tag "role"）
    - 定义 Content 字段（string 类型，JSON tag "content"）
    - _Requirements: 4.1_
  
  - [x] 2.2 在 types.go 中定义 InvokeRequest 结构体
    - 定义 System 字段（string 类型，JSON tag "system"）
    - 定义 Messages 字段（[]Message 类型，JSON tag "messages"）
    - _Requirements: 4.2_
  
  - [x] 2.3 在 types.go 中定义 InvokeResponse 结构体
    - 定义 Answer 字段（string 类型，JSON tag "answer"）
    - _Requirements: 4.3_
  
  - [x] 2.4 在 types.go 中定义 ClaudeOutput 结构体
    - 定义 Result 字段（string 类型，JSON tag "result"）
    - 添加其他可选字段（type, total_cost_usd, duration_ms）
    - _Requirements: 3.3_

- [x] 3. 实现 Claude CLI 调用模块
  - [x] 3.1 创建 claude.go 文件并实现 buildPrompt 函数
    - 接收 []Message 参数
    - 遍历 messages 数组
    - 根据 role 字段添加 "User:" 或 "Assistant:" 前缀
    - 使用换行符连接所有消息
    - 返回拼接后的字符串
    - _Requirements: 2.1, 2.2, 2.3, 4.4_
  
  - [x] 3.2 在 claude.go 中实现 runClaude 函数
    - 接收 prompt 和 systemPrompt 两个字符串参数
    - 构建命令参数数组：["--print", prompt, "--output-format", "json", "--allowedTools", "WebSearch"]
    - 如果 systemPrompt 非空，追加 ["--append-system-prompt", systemPrompt]
    - 使用 exec.Command("claude", args...) 执行命令
    - 捕获命令输出（使用 CombinedOutput）
    - 检查命令执行错误，如果失败返回错误信息
    - 解析 JSON 输出到 ClaudeOutput 结构体
    - 返回 Result 字段和可能的错误
    - _Requirements: 2.4, 2.5, 3.1, 3.2, 3.3, 3.4, 4.5_

- [x] 4. 实现 HTTP handler
  - [x] 4.1 创建 handler.go 文件并实现 handleInvoke 函数
    - 检查 HTTP 方法是否为 POST，否则返回 405
    - 解析请求体 JSON 到 InvokeRequest 结构体
    - 如果解析失败，返回 400 错误
    - 调用 buildPrompt 函数构建 prompt
    - 调用 runClaude 函数执行 Claude CLI
    - 如果 runClaude 返回错误，返回 500 错误响应
    - 如果成功，构建 InvokeResponse 并返回 200 响应
    - 设置响应头 Content-Type 为 application/json
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.4, 3.5_

- [x] 5. 实现程序入口
  - [x] 5.1 创建 main.go 文件并实现 main 函数
    - 使用 http.HandleFunc 注册 "/invoke" 路由到 handleInvoke
    - 打印启动日志 "Gateway service starting on :8080"
    - 调用 http.ListenAndServe(":8080", nil) 启动服务器
    - 使用 log.Fatal 包装以处理启动错误
    - _Requirements: 1.1, 1.5_

- [x] 6. 实现简化的聊天接口
  - [x] 6.1 在 types.go 中定义 ChatRequest 结构体
    - 定义 Prompt 字段（string 类型，JSON tag "prompt"）
    - 定义 System 字段（string 类型，JSON tag "system"）
    - _Requirements: 5.2, 5.3_
  
  - [x] 6.2 在 handler.go 中实现 handleChat 函数
    - 检查 HTTP 方法是否为 POST，否则返回 405
    - 解析请求体 JSON 到 ChatRequest 结构体
    - 如果解析失败，返回 400 错误
    - 直接使用 ChatRequest.Prompt 作为用户输入
    - 调用 runClaude 函数执行 Claude CLI（传入 prompt 和 system）
    - 如果 runClaude 返回错误，返回 500 错误响应
    - 如果成功，构建 InvokeResponse 并返回 200 响应
    - 设置响应头 Content-Type 为 application/json
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  
  - [x] 6.3 在 main.go 中注册 /chat 路由
    - 在 main 函数中添加 http.HandleFunc("/chat", handleChat)
    - 确保在启动服务器之前注册路由
    - _Requirements: 5.1_

- [x] 7. 支持多种 CLI 工具（Claude 和 Codex）
  - [x] 7.1 在 types.go 中为 InvokeRequest 和 ChatRequest 添加 CLI 字段
    - 在 InvokeRequest 中添加 CLI 字段（string 类型，JSON tag "cli"）
    - 在 ChatRequest 中添加 CLI 字段（string 类型，JSON tag "cli"）
    - _Requirements: 6.1_
  
  - [x] 7.2 重构 claude.go 中的 runClaude 函数为 runCLI
    - 将函数名从 runClaude 改为 runCLI
    - 添加 cliName 参数作为第一个参数（string 类型）
    - 如果 cliName 为空，默认设置为 "claude"
    - 使用 cliName 变量替代硬编码的 "claude" 命令
    - 保持其他逻辑不变（参数构建、环境变量、JSON 解析）
    - _Requirements: 6.2, 6.3, 6.4, 6.5_
  
  - [x] 7.3 更新 handler.go 中的调用
    - 在 handleInvoke 中，将 runClaude 调用改为 runCLI，传入 req.CLI 作为第一个参数
    - 在 handleChat 中，将 runClaude 调用改为 runCLI，传入 req.CLI 作为第一个参数
    - _Requirements: 6.2, 6.3_

- [x] 8. 在配置文件中支持 CLI 字段
  - [x] 8.1 在 config.go 的 ProfileConfig 结构体中添加 CLI 字段
    - 添加 CLI 字段（string 类型，JSON tag "cli"）
    - _Requirements: 7.1_
  
  - [x] 8.2 更新 runCLI 函数以支持从 profile 读取 CLI 配置
    - 修改 runCLI 函数，在 cliName 为空时，尝试从 profile 获取 CLI 设置
    - 如果 profile 有 CLI 字段且 cliName 为空，使用 profile.CLI
    - 如果都没有，默认使用 "claude"
    - 添加日志输出，显示最终使用的 CLI 工具和来源（请求/profile/默认）
    - _Requirements: 7.2, 7.3, 7.4, 7.5_
