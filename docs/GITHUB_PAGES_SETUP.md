# GitHub Pages 配置指南

## 🎯 完整配置步骤

### 步骤 1: 配置 GitHub Actions 权限

首先需要给 GitHub Actions 足够的权限来创建和推送到 gh-pages 分支。

1. 进入你的 GitHub 仓库
2. 点击 **Settings**（设置）
3. 在左侧菜单找到 **Actions** > **General**
4. 滚动到 **Workflow permissions** 部分
5. 选择 **Read and write permissions**
6. 勾选 **Allow GitHub Actions to create and approve pull requests**
7. 点击 **Save**

![Workflow Permissions](https://docs.github.com/assets/cb-45061/images/help/repository/actions-workflow-permissions-repository.png)

### 步骤 2: 手动触发首次 GitHub Actions 运行

gh-pages 分支需要先被创建，然后才能配置 Pages。

1. 进入仓库的 **Actions** 标签
2. 在左侧找到 **Generate Release Notes** 工作流
3. 点击右侧的 **Run workflow** 按钮
4. 选择 `main` 分支
5. 点击绿色的 **Run workflow** 按钮

![Run Workflow](https://docs.github.com/assets/cb-35844/images/help/actions/workflow-dispatch-button.png)

6. 等待工作流运行完成（约 1-2 分钟）
7. 刷新页面，查看运行结果
   - ✅ 绿色勾号 = 成功
   - ❌ 红色叉号 = 失败（点击查看日志）

### 步骤 3: 验证 gh-pages 分支已创建

1. 在仓库主页，点击分支下拉菜单（默认显示 `main`）
2. 查看是否有 `gh-pages` 分支
3. 或者访问：`https://github.com/<username>/<repo>/tree/gh-pages`

如果看到 gh-pages 分支，说明 Actions 运行成功！

### 步骤 4: 配置 GitHub Pages

现在可以配置 Pages 使用 gh-pages 分支了。

1. 进入仓库的 **Settings**
2. 在左侧菜单找到 **Pages**
3. 在 **Source** 部分：
   - **Branch**: 选择 `gh-pages`
   - **Folder**: 选择 `/ (root)`
4. 点击 **Save**

![GitHub Pages Settings](https://docs.github.com/assets/cb-47267/images/help/pages/select-branch.png)

5. 等待几分钟，页面会显示：
   ```
   Your site is live at https://<username>.github.io/<repo>/
   ```

### 步骤 5: 访问你的页面

打开浏览器，访问：
```
https://<username>.github.io/<repo>/
```

你应该能看到 Release Notes 页面！

## 🔍 故障排除

### 问题 1: Actions 运行失败

**错误信息**: `Permission denied` 或 `403 Forbidden`

**解决方案**:
- 确认步骤 1 中的权限设置正确
- 检查 `GITHUB_TOKEN` 是否有效（通常自动提供）

### 问题 2: gh-pages 分支未创建

**可能原因**:
- Actions 运行失败
- 权限不足

**解决方案**:
1. 查看 Actions 运行日志
2. 确认权限设置
3. 重新运行工作流

### 问题 3: 页面显示 404

**可能原因**:
- Pages 未正确配置
- 分支选择错误
- 需要等待部署完成

**解决方案**:
1. 确认 Pages 设置中选择了 `gh-pages` 分支
2. 等待 3-5 分钟让 GitHub 部署
3. 清除浏览器缓存
4. 检查 gh-pages 分支是否有 `index.html` 文件

### 问题 4: 页面显示 README.md 而不是 Release Notes

**可能原因**:
- Pages 配置选择了错误的分支（main 而不是 gh-pages）
- gh-pages 分支中没有 index.html

**解决方案**:
1. 确认 Pages 设置选择的是 `gh-pages` 分支
2. 访问 `https://github.com/<username>/<repo>/tree/gh-pages`
3. 确认有 `index.html` 文件
4. 如果没有，重新运行 Actions 工作流

### 问题 5: 数据未更新

**可能原因**:
- Actions 未按计划运行
- 浏览器缓存

**解决方案**:
1. 检查 Actions 标签，查看最近的运行时间
2. 手动触发一次工作流
3. 清除浏览器缓存（Ctrl+Shift+R 或 Cmd+Shift+R）
4. 查看页面底部的 "Last updated" 时间

## 📊 验证清单

完成配置后，使用这个清单验证：

- [ ] GitHub Actions 权限设置为 "Read and write"
- [ ] 首次工作流运行成功（绿色勾号）
- [ ] gh-pages 分支已创建
- [ ] gh-pages 分支包含 index.html 文件
- [ ] Pages 设置选择了 gh-pages 分支
- [ ] 页面 URL 显示为 "live"
- [ ] 可以访问页面并看到 Release Notes
- [ ] 页面显示 5 个 CLI 标签（Claude, Codex, Cursor, Gemini, Qwen）

## 🔄 自动更新

配置完成后，页面将：
- ⏰ 每小时自动更新一次
- 🔄 代码变更时自动更新
- 🖱️ 可以手动触发更新

## 📱 访问方式

### 公开访问
```
https://<username>.github.io/<repo>/
```

### 自定义域名（可选）

如果你有自定义域名：

1. 在 Pages 设置中添加自定义域名
2. 在域名提供商处添加 CNAME 记录
3. 取消注释工作流中的 CNAME 配置：

```yaml
# 在 .github/workflows/release-notes.yml 中
echo "your-domain.com" > gh-pages/CNAME
```

## 🎉 完成！

现在你的 Release Notes 页面已经：
- ✅ 自动部署到 GitHub Pages
- ✅ 每小时自动更新
- ✅ 公开可访问
- ✅ 无需维护服务器

享受自动化的便利吧！🚀

## 📚 相关资源

- [GitHub Pages 官方文档](https://docs.github.com/en/pages)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [peaceiris/actions-gh-pages](https://github.com/peaceiris/actions-gh-pages)
