# 更新日志

## v2.0.0 - 多配置支持

### 新功能

- ✨ **多配置支持**: 支持同时配置多个 Claude API 提供商（MiniMax、智谱 GLM、Kimi 等）
- 🔧 **配置文件**: 通过 `configs.json` 管理所有配置
- 🎯 **动态切换**: 请求时可通过 `profile` 字段指定使用的配置
- 📝 **详细日志**: 记录使用的 profile 和配置信息

### 配置文件

创建 `configs.json`（敏感信息建议使用 `.env` 变量占位符）：

```json
{
  "profiles": {
    "minimax": {
      "name": "MiniMax",
      "env": { ... }
    },
    "glm": {
      "name": "智谱 GLM",
      "env": { ... }
    }
  },
  "default": "minimax"
}
```

### API 变更

请求格式新增 `profile` 字段：

```json
{
  "profile": "minimax",  // 新增：指定配置
  "system": "你是一个助手",
  "messages": [...]
}
```

### 使用示例

```bash
# 使用 MiniMax
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{"profile": "minimax", "messages": [...]}'

# 使用智谱 GLM
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{"profile": "glm", "messages": [...]}'
```

---

## v1.0.0 - 初始版本

### 功能

- HTTP 到 Claude CLI 的桥接
- 支持对话历史
- 支持 system prompt
- 启用 WebSearch 工具
- 日志记录到文件
- 性能统计
