# GitHub Actions 定时爬虫配置说明

## 功能说明

本项目使用 GitHub Actions 实现每日自动爬取报纸 PDF 文件并提交到仓库。

## 工作流程

1. **定时执行**：每天 UTC 0:00（北京时间早上 8:00）自动运行
2. **手动触发**：可以在 GitHub Actions 页面手动运行
3. **自动提交**：生成的 PDF 文件会自动提交到 `web/files/` 目录

## 使用方法

### 1. 启用 GitHub Actions

确保你的仓库已启用 GitHub Actions：
- 进入仓库的 **Settings** → **Actions** → **General**
- 确保 **Actions permissions** 设置为允许运行

### 2. 设置仓库权限

为了让 GitHub Actions 能够推送代码，需要设置权限：
- 进入仓库的 **Settings** → **Actions** → **General**
- 滚动到 **Workflow permissions** 部分
- 选择 **Read and write permissions**
- 勾选 **Allow GitHub Actions to create and approve pull requests**
- 点击 **Save**

### 3. 手动触发测试

首次配置后，建议手动测试一次：
1. 进入仓库的 **Actions** 标签页
2. 选择 **Daily Papers Crawler** 工作流
3. 点击 **Run workflow** 按钮
4. 选择分支（通常是 `main`）
5. 点击绿色的 **Run workflow** 按钮

### 4. 查看执行结果

- 在 **Actions** 标签页可以看到所有的执行记录
- 点击具体的运行记录可以查看详细日志
- 生成的 PDF 文件会在 **Artifacts** 中保存 30 天

## 定时设置说明

当前设置为每天 UTC 0:00 执行（北京时间早上 8:00）。

如需修改执行时间，编辑 `.github/workflows/daily-papers.yml` 文件中的 cron 表达式：

```yaml
schedule:
  - cron: '0 0 * * *'  # 分 时 日 月 星期（UTC时间）
```

常用时间示例：
- `'0 0 * * *'` - 每天 UTC 0:00（北京时间 8:00）
- `'0 12 * * *'` - 每天 UTC 12:00（北京时间 20:00）
- `'0 0,12 * * *'` - 每天 UTC 0:00 和 12:00 各执行一次
- `'0 0 * * 1-5'` - 每周一到周五 UTC 0:00 执行

## 注意事项

1. **时区**：GitHub Actions 使用 UTC 时间，北京时间 = UTC + 8 小时
2. **执行延迟**：定时任务可能会有几分钟的延迟
3. **依赖项**：工作流会自动安装 Go 依赖，无需额外配置
4. **错误处理**：每个报纸的爬取使用 `continue-on-error: true`，某个报纸失败不会影响其他报纸
5. **存储限制**：GitHub 仓库有存储限制，定期清理旧的 PDF 文件

## 文件说明

- `.github/workflows/daily-papers.yml` - GitHub Actions 工作流配置文件
- `web/files/` - PDF 文件存储目录
- 生成的文件格式：`rmrb_YYYYMMDD.pdf`、`zgcsb_YYYYMMDD.pdf` 等

## 查看历史文件

所有提交的 PDF 文件都会保存在 Git 历史记录中，可以通过以下方式查看：
- 在 GitHub 网页端浏览 `web/files/` 目录
- 使用 `git log -- web/files/` 查看文件变更历史
- 通过 Git 命令检出历史版本的文件
