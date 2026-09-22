# Migrations

本项目使用 GORM AutoMigrate 在服务启动时自动建表/迁移（见 `backend/internal/database/database.go`）。

实体表：
- users
- papers
- reviews
- revisions
- plagiarism_checks
- withdrawals
- audit_logs

## 撤稿功能迁移说明（AutoMigrate 自动完成）

- `papers.status` 新增终态值 `withdrawn`（已撤稿）；状态列为 varchar，无需 DDL 改类型，已撤稿论文在 repository 统计查询中显式排除。
- `reviews.status` 新增值 `closed`：撤稿批准时，论文下 `invited`/`accepted` 的未完成审稿被批量置为 `closed`，`completed`/`declined` 历史记录原样保留。
- 新表 `withdrawals`：撤稿申请（原因、替代处理说明、状态 pending/approved/rejected、处理意见/处理人/处理时间）。
- 部分唯一索引 `uniq_withdrawal_pending`：`CREATE UNIQUE INDEX ... ON withdrawals (paper_id) WHERE status = 'pending'`，
  与 service 层事务 + `SELECT ... FOR UPDATE` 共同保证重复或并发申请只保留一条待处理记录；驳回后旧申请终态化，作者可再次申请。

如需手工执行 SQL 迁移，可参照 database/database.go 中的 DSN 与模型结构编写。
