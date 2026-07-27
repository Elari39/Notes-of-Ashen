# 运维手册（备份、恢复与发布）

本手册补齐审计报告（[PROJECT-REVIEW.md](PROJECT-REVIEW.md) §4 P2/P3）指出的备份持久化、异地保存和恢复演练运维闭环。代码层已有的保护（age 加密导出、SHA-256 校验、恢复锁、迁移 checksum 等）不能替代下述外部备份策略。

## 1. 备份策略

### 1.1 每日自动备份（scripts/backup.ps1）

```powershell
# 默认输出到 ./backups/<时间戳>/，保留 14 天
pwsh scripts/backup.ps1

# 自定义输出目录与保留周期
pwsh scripts/backup.ps1 -OutputDir D:\noa-backups -RetentionDays 30

# 仅数据库、跳过媒体卷
pwsh scripts/backup.ps1 -SkipMedia
```

每次备份产物：

| 文件 | 内容 |
| --- | --- |
| `mysql-notes_of_ashen.sql.gz` | `mysqldump --single-transaction --routines --triggers` 全库导出 |
| `media.tar.gz` | `goblog_media_data` 媒体数据卷归档 |
| `SHA256SUMS.txt` | 以上文件的 SHA-256 清单（LF 格式，Linux 端可直接 `sha256sum -c`） |

挂接 Windows 计划任务（每日 03:00 示例）：

```powershell
Register-ScheduledTask -TaskName "noa-backup" `
  -Action (New-ScheduledTaskAction -Execute "pwsh" -Argument "-NoProfile -File F:\WorkSpace\Coding\Go\GoBlog\scripts\backup.ps1") `
  -Trigger (New-ScheduledTaskTrigger -Daily -At 03:00)
```

### 1.2 周期性加密全量包（后台系统工具）

`scripts/backup.ps1` 覆盖数据库与媒体文件，但不含搜索索引重建标记与应用级关联校验。每周（或每次重大变更前）额外通过管理后台「系统工具」导出 age scrypt 加密的 `.noa-backup` 全量包，它具备应用层完整性校验，是恢复演练的标准输入。

### 1.3 保留与容量建议

- 本地 `backups/`：至少保留 14 天（脚本默认），磁盘紧张时优先缩短媒体归档保留期。
- `.noa-backup` 加密全量包：至少保留最近 4 份（约一个月）。
- 发布前必须先完成一次备份（含数据库迁移前，见 README 迁移说明）。

## 2. 异地副本

本地备份无法抵御主机、磁盘或数据卷同时损坏，必须建立异地副本：

- 使用 rclone / 对象存储（S3、OSS、B2 等）同步 `backups/` 与 `.noa-backup` 文件，示例：
  `rclone sync F:\WorkSpace\Coding\Go\GoBlog\backups remote:noa-backups --checksum`
- 上传公有云前确认内容已加密（`.noa-backup` 自带 age 加密；`backups/` 目录为明文，建议先用 age/7z 加密再上传）。
- 同步任务失败需要有告警（计划任务失败通知、rclone 日志检查或云端对象数量监控）。

### 密钥离线保管

以下密钥丢失将导致备份不可恢复或会话/AI 密文全部失效，必须离线保存在密码管理器或纸质介质，不入库、不上传：

- `.noa-backup` 的 age 导出口令。
- `.env` 中的 `APP_AUTH_ACCESS_SECRET`（AI API Key 密文的派生加密密钥；轮换前先阅读 README 迁移说明）。
- MySQL root/业务账号密码、Redis/RabbitMQ/Meilisearch 凭据。

## 3. 恢复演练（月度）

至少每月在隔离环境（另一台机器或临时 Compose 项目，禁止直接在生产栈上演练）执行一次全量恢复：

1. 记录起始时间、使用的备份版本（时间戳/tag/git commit）。
2. 以 `.env.example` 为基础创建隔离环境 `.env`（新的项目名与端口，`APP_AUTH_COOKIE_SECURE=false`）。
3. 启动隔离栈，通过后台「系统工具」导入 `.noa-backup` 加密包执行恢复；或按 `backups/` 产物手动恢复：`gunzip < mysql-*.sql.gz | docker compose exec -T mysql mysql ...`，媒体 tar 解回 `goblog_media_data` 卷。
4. 验收清单（全部通过才算演练成功）：
   - [ ] 管理员登录、刷新页面会话保持；
   - [ ] 文章列表/详情、分类、标签、归档数据完整；
   - [ ] 媒体文件可访问，上传新文件正常；
   - [ ] 启用搜索时索引重建完成、搜索可用；
   - [ ] AI 设置存在时密钥密文可用（必要时按 README 重新录入）；
   - [ ] `/rss.xml`、`/sitemap.xml` 正常。
   - [ ] 生产环境站点设置中的 `siteBaseUrl` 为正式 HTTPS 域名，RSS、Sitemap 和文章分享链接均使用该域名。
5. 记录结束时间、总耗时、发现的问题；耗时应在可接受的目标恢复时间（建议 ≤ 2 小时）内。
6. 演练完成后销毁隔离栈（`docker compose -p <隔离项目名> down -v`）。

演练记录（日期、备份版本、耗时、结论）建议追加保存在私有运维笔记中，形成可审计的历史。

## 4. 发布与回滚

使用 [scripts/release.ps1](../scripts/release.ps1) 以不可变镜像 tag 发布（详见 README「发布与回滚」）：

- 发布：`pwsh scripts/release.ps1`（构建 → 更新 `.env` IMAGE_TAG → `up -d` → 追加记录）。
- 代码回退：`pwsh scripts/release.ps1 -Rollback <旧tag>`。脚本默认校验目标镜像迁移版本不低于实际数据库版本；不兼容时会拒绝并要求恢复备份或选择兼容镜像。
- 紧急绕过：仅在已验证备份恢复路径后使用 `pwsh scripts/release.ps1 -Rollback <旧tag> -AllowIncompatibleSchema`。该操作不会回滚数据库 schema，记录中的 action 为 `code-rollback`。
- 正式发布默认要求工作区干净且 tag 未被使用；若必须发布未提交内容，显式使用 `-AllowDirty`，记录会标明不可复现风险。
- 每次发布/代码回退会在 `deploy/release-history.local.jsonl` 记录时间、tag、git commit、工作区状态、镜像 ID 和可用的 RepoDigest；该文件不入库，请随备份一起异地保存。
- 发布顺序建议：备份 → 发布 → 按 README「生产发布验收清单」验收 → 失败则回滚并恢复备份。
