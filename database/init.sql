-- PaperFlow 数据库初始化脚本（可选，服务启动时会用 GORM AutoMigrate 自动建表）
-- 表：users / papers / reviews / revisions / plagiarism_checks / withdrawals / audit_logs

-- papers.status 新增终态枚举值 'withdrawn'（已撤稿），已撤稿论文从论文库与统计排除。
-- reviews.status 新增 'closed'（论文撤稿批准后未完成审稿被系统关闭，记录保留可查看）。
-- withdrawals：撤稿申请，(paper_id) 上存在部分唯一索引，仅允许每篇论文一条 status='pending' 的申请。

-- 演示账号（密码为 bcrypt 哈希，服务启动 Seed 阶段自动写入，无需手工执行）
-- admin/admin123、editor/editor123、reviewer/reviewer123、author/author123
