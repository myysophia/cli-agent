# Claude Skills 使用指南

## 什么是 Claude Skills？

Claude Skills 是 Claude CLI 的一个强大功能，允许 Claude 访问本地文件和目录。通过 Skills，Claude 可以：

- 读取本地文档作为上下文
- 理解你的代码库结构
- 基于你的研究报告提供专业建议
- 访问项目文档和知识库

## 配置 Skills

### 基本配置

在 `configs.json` 中为 profile 添加 `skills` 字段：

```json
{
  "profiles": {
    "qwen-with-reports": {
      "name": "Qwen with Research Reports",
      "cli": "claude",
      "skills": ["./reporter"],
      "env": {
        "ANTHROPIC_API_KEY": "your-api-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

### 多个 Skills

你可以配置多个 skill 路径：

```json
{
  "skills": [
    "./reporter",           // 研究报告目录
    "./docs",              // 文档目录
    "./knowledge-base",    // 知识库
    "./research.pdf"       // 单个文件
  ]
}
```

### Skills 路径类型

**目录路径**：
```json
"skills": ["./reporter"]
```
- Claude 会递归读取目录下的所有文件
- 支持相对路径和绝对路径

**单个文件**：
```json
"skills": ["./docs/important-report.md"]
```
- 只读取指定的单个文件

**混合使用**：
```json
"skills": [
  "./reporter",                    // 整个目录
  "./docs/summary.md",            // 单个文件
  "/absolute/path/to/research"    // 绝对路径
]
```

## 使用场景

### 1. 研究报告分析

**配置**：
```json
{
  "profiles": {
    "research-assistant": {
      "name": "Research Assistant",
      "cli": "claude",
      "skills": ["./reporter", "./papers"],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

**使用**：
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "根据研究报告，总结最新的 AI 技术趋势",
    "profile": "research-assistant"
  }'
```

### 2. 代码库理解

**配置**：
```json
{
  "profiles": {
    "code-reviewer": {
      "name": "Code Reviewer",
      "cli": "claude",
      "skills": ["./src", "./docs/architecture.md"],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

**使用**：
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "分析这个项目的架构设计，有什么可以改进的地方？",
    "profile": "code-reviewer"
  }'
```

### 3. 文档问答

**配置**：
```json
{
  "profiles": {
    "doc-qa": {
      "name": "Documentation Q&A",
      "cli": "claude",
      "skills": ["./docs", "./README.md", "./API.md"],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

**使用**：
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "如何配置这个项目？",
    "profile": "doc-qa"
  }'
```

### 4. 知识库助手

**配置**：
```json
{
  "profiles": {
    "knowledge-base": {
      "name": "Knowledge Base Assistant",
      "cli": "claude",
      "skills": [
        "./knowledge-base/tech",
        "./knowledge-base/business",
        "./knowledge-base/processes"
      ],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  }
}
```

## 最佳实践

### 1. 组织你的文件

```
project/
├── reporter/              # 研究报告
│   ├── 2024-q1-report.md
│   ├── 2024-q2-report.md
│   └── analysis/
│       └── trend-analysis.md
├── docs/                  # 项目文档
│   ├── architecture.md
│   └── api-reference.md
└── knowledge-base/        # 知识库
    ├── tech/
    └── business/
```

### 2. 使用描述性的 Profile 名称

```json
{
  "profiles": {
    "qwen-with-reports": {
      "name": "Qwen with Research Reports",
      // ...
    },
    "qwen-with-docs": {
      "name": "Qwen with Project Docs",
      // ...
    }
  }
}
```

### 3. 根据任务选择合适的 Skills

- **技术问题**：包含代码库和技术文档
- **业务分析**：包含研究报告和业务文档
- **综合咨询**：包含多个领域的文档

### 4. 控制 Skills 范围

不要包含过多无关文件，这会：
- 增加处理时间
- 可能导致上下文混乱
- 浪费 token

**推荐**：
```json
"skills": ["./reporter/2024"]  // 只包含 2024 年的报告
```

**不推荐**：
```json
"skills": ["./"]  // 包含整个项目（可能有很多无关文件）
```

## 技术细节

### 命令行参数

网关会将 skills 转换为 Claude CLI 的 `--add-dir` 参数：

```bash
claude --print "your prompt" \
  --output-format json \
  --allowedTools WebSearch \
  --add-dir ./reporter \
  --add-dir ./docs
```

### 日志输出

启用 Skills 后，日志会显示：

```
📚 Using 2 skill(s): [./reporter ./docs]
```

### 支持的文件类型

Claude Skills 支持多种文件格式：
- Markdown (`.md`)
- 文本文件 (`.txt`)
- 代码文件 (`.py`, `.js`, `.go`, 等)
- PDF 文件 (`.pdf`)
- 其他文本格式

## 故障排查

### Skills 路径不存在

**错误**：Claude CLI 可能报错找不到路径

**解决**：
- 确认路径存在
- 使用相对于网关启动目录的路径
- 或使用绝对路径

### Skills 文件过大

**问题**：处理时间过长或超时

**解决**：
- 减少 skills 数量
- 使用更具体的路径
- 分割大文件

### 权限问题

**错误**：无法读取文件

**解决**：
- 确认文件权限
- 确认网关进程有读取权限

## 示例：完整配置

```json
{
  "profiles": {
    "kimi": {
      "name": "Kimi",
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.kimi.com/coding/",
        "ANTHROPIC_AUTH_TOKEN": "your-token"
      }
    },
    "qwen-basic": {
      "name": "Qwen Basic",
      "cli": "claude",
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    },
    "qwen-with-reports": {
      "name": "Qwen with Research Reports",
      "cli": "claude",
      "skills": ["./reporter"],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    },
    "qwen-full-context": {
      "name": "Qwen with Full Context",
      "cli": "claude",
      "skills": [
        "./reporter",
        "./docs",
        "./knowledge-base"
      ],
      "env": {
        "ANTHROPIC_API_KEY": "your-key",
        "ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic",
        "ANTHROPIC_MODEL": "qwen3-max"
      }
    }
  },
  "default": "qwen-basic"
}
```

## 注意事项

1. **仅 Claude CLI 支持**：Skills 功能仅在使用 Claude CLI 时有效，Codex CLI 不支持
2. **路径安全**：确保 skills 路径不包含敏感信息
3. **性能考虑**：大量文件会增加处理时间和 token 消耗
4. **参数说明**：网关使用 `--add-dir` 参数来添加目录访问权限
5. **权限模式**：Claude 会请求访问这些目录的权限，在 `--print` 模式下会自动授权

## 相关资源

- [Claude CLI 官方文档](https://docs.anthropic.com/claude/docs/claude-cli)
- [项目 README](./README.md)
- [配置示例](./configs.example.json)
