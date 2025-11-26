# GitHub Pages 部署指南

本项目使用 GitHub Actions 自动生成 CLI Release Notes 的静态 HTML 页面，并部署到 GitHub Pages。

## 🚀 功能特性

- ⏰ **自动更新**：每小时自动获取最新的 release notes
- 📄 **静态页面**：生成纯静态 HTML，无需后端服务器
- 🎨 **美观界面**：响应式设计，支持移动端
- 🔄 **手动触发**：支持通过 GitHub Actions 手动触发更新

## 📋 支持的 CLI 工具

- Claude CLI
- Codex CLI
- Cursor Agent
- Gemini CLI
- Qwen CLI

## 🛠️ 设置步骤

### 1. 启用 GitHub Pages

1. 进入仓库的 **Settings** > **Pages**
2. 在 **Source** 下选择 **Deploy from a branch**
3. 选择 **gh-pages** 分支
4. 点击 **Save**

### 2. 配置 GitHub Actions 权限

1. 进入仓库的 **Settings** > **Actions** > **General**
2. 在 **Workflow permissions** 下选择 **Read and write permissions**
3. 勾选 **Allow GitHub Actions to create and approve pull requests**
4. 点击 **Save**

### 3. 运行 GitHub Action

GitHub Action 会在以下情况自动运行：

- **定时任务**：每小时自动运行一次
- **代码推送**：当 `internal/release_notes/` 或 `web/templates/` 目录有更新时
- **手动触发**：在 Actions 页面手动运行

首次设置后，可以手动触发一次：

1. 进入仓库的 **Actions** 标签
2. 选择 **Generate Release Notes** workflow
3. 点击 **Run workflow** > **Run workflow**

### 4. 访问页面

部署完成后，可以通过以下 URL 访问：

```
https://<your-username>.github.io/<repository-name>/
```

例如：`https://ninesun.github.io/dify-cli-gateway/`

## 🔧 本地测试

在推送到 GitHub 之前，可以在本地测试 HTML 生成：

```bash
# 运行生成脚本
./scripts/generate-release-notes.sh

# 在浏览器中打开生成的 HTML
open release-notes.html  # macOS
xdg-open release-notes.html  # Linux
start release-notes.html  # Windows
```

## 📁 相关文件

- `.github/workflows/release-notes.yml` - GitHub Actions 工作流配置
- `cmd/generate-html/main.go` - HTML 生成器
- `web/templates/release_notes_static.html` - 静态 HTML 模板
- `scripts/generate-release-notes.sh` - 本地测试脚本

## 🔄 更新频率

- **自动更新**：每小时一次（可在 `.github/workflows/release-notes.yml` 中修改 cron 表达式）
- **手动更新**：随时可以在 Actions 页面手动触发

## 🎨 自定义

### 修改更新频率

编辑 `.github/workflows/release-notes.yml` 中的 cron 表达式：

```yaml
schedule:
  - cron: '0 * * * *'  # 每小时
  # - cron: '0 */2 * * *'  # 每 2 小时
  # - cron: '0 0 * * *'  # 每天午夜
```

### 自定义域名

如果你有自定义域名，可以在工作流中取消注释 CNAME 配置：

```yaml
# 在 .github/workflows/release-notes.yml 中
echo "your-domain.com" > gh-pages/CNAME
```

### 修改样式

编辑 `web/templates/release_notes_static.html` 中的 CSS 样式。

## 🐛 故障排除

### 页面显示 404

1. 确认 GitHub Pages 已启用
2. 确认选择了 `gh-pages` 分支
3. 等待几分钟让 GitHub 部署完成

### Action 运行失败

1. 检查 Actions 日志查看错误信息
2. 确认 Workflow permissions 设置正确
3. 确认代码可以正常编译（运行 `go build ./cmd/generate-html`）

### 数据未更新

1. 检查 Action 是否成功运行
2. 查看 `last-update.txt` 文件确认更新时间
3. 清除浏览器缓存后重新访问

## 📊 监控

可以在以下位置查看部署状态：

- **Actions 标签**：查看工作流运行历史
- **gh-pages 分支**：查看生成的静态文件
- **last-update.txt**：查看最后更新时间

## 🔗 相关链接

- [GitHub Pages 文档](https://docs.github.com/en/pages)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Cron 表达式参考](https://crontab.guru/)
