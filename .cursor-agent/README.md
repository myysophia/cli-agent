# Cursor Agent CI 配置

本目录包含 Cursor Agent 在 GitHub Actions 中运行的配置文件。

## 📁 文件说明

- `permissions.json` - 权限配置，限制 Agent 的操作范围
- `prompts/` - 预定义的测试提示（可选）

## 🔐 权限配置

当前配置为**只读模式**，Agent 只能：
- ✅ 读取所有文件
- ✅ 运行只读 shell 命令（go, grep, find, ls, cat 等）
- ❌ 不能写入文件
- ❌ 不能执行 git 操作
- ❌ 不能修改配置文件

## 🧪 测试场景

GitHub Actions 工作流会运行以下测试：

1. **代码分析** - 分析项目结构和主要包
2. **Release Notes 检查** - 检查支持的 CLI 工具
3. **API 端点审查** - 列出所有 HTTP API 端点
4. **MCP 工具测试** - 使用 fetch 工具获取外部数据

## 🔧 自定义测试

可以通过手动触发工作流并提供自定义提示来运行额外测试：

1. 进入 Actions 标签
2. 选择 "Cursor Agent Test" 工作流
3. 点击 "Run workflow"
4. 在 "test_prompt" 字段输入自定义提示
5. 点击 "Run workflow"

## 📊 查看结果

测试结果会：
- 显示在 GitHub Actions 的 Summary 中
- 作为 Artifact 上传，保留 30 天
- 包含详细的输出日志

## ⚠️ 注意事项

- 需要在仓库 Secrets 中配置 `CURSOR_API_KEY`
- 测试会消耗 Cursor API 配额
- 建议在非高峰时段运行
