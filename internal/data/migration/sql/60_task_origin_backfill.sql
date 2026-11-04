-- 任务面板重写（pkg/task 记录触发方式、开始时间、步骤与日志）后，为历史任务行回填新字段：
--   1. 定时单例记为「计划」触发，有发起用户的一次性任务记为「手动」触发，其余保持默认「系统」；
--   2. 已开始过的任务以创建时间作为开始时间（历史行没有单独记录开始时间）。
-- 标识符使用双引号，MySQL 下由执行器改写为反引号。

UPDATE "tasks" SET "trigger_type" = 'schedule' WHERE "kind" = 'scheduled';
UPDATE "tasks" SET "trigger_type" = 'manual' WHERE "kind" <> 'scheduled' AND "user_id" <> '';
UPDATE "tasks" SET "start_time" = "create_time" WHERE "start_time" IS NULL AND "status" <> 'pending';
