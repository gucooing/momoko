-- 移除 sub2api 模块后清理数据库残留：
--   1. 数据表 sub2api_* 与生图 image_gen_*（ent 自动迁移不会删除已不在 schema 中的表）；
--   2. configs 表中的 sub2api.* 配置键（含 admin_api_key）；
--   3. tasks 表中 sub2api.* 类型的任务记录（如 sub2api.lottery.tick 定时任务）。
-- 标识符使用双引号，MySQL 下由执行器改写为反引号。

-- sub2api_usage_records 外键引用 sub2api_groups，先删子表。
DROP TABLE IF EXISTS "sub2api_usage_records";
DROP TABLE IF EXISTS "sub2api_groups";
DROP TABLE IF EXISTS "sub2api_lottery_participants";
DROP TABLE IF EXISTS "sub2api_lottery_rounds";
DROP TABLE IF EXISTS "sub2api_announcements";
DROP TABLE IF EXISTS "sub2api_timeline_items";
DROP TABLE IF EXISTS "image_gen_images";
DROP TABLE IF EXISTS "image_gen_generations";

DELETE FROM "configs" WHERE "key" LIKE 'sub2api.%';
DELETE FROM "tasks" WHERE "type" LIKE 'sub2api.%';
