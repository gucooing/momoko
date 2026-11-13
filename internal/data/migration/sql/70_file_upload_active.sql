-- 上传会话的唯一索引改为 (source_id, path, hash, active)：MySQL 不支持部分索引，
-- 原先「仅约束未完成且未取消」的条件在 MySQL 上不生效，已收尾的会话会挡住同一文件再次上传。
-- 自动迁移新增的 active 列默认为空，这里把存量进行中的会话补记为 true，使其仍受唯一约束且能续传。
UPDATE "file_uploads" SET "active" = TRUE WHERE NOT "completed" AND NOT "cancel";
