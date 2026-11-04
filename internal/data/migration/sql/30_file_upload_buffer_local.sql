-- 分片上传会话新增 buffer_local（分片是否缓冲在目标目录旁）：此前以 source_id 为空判定本地缓冲，
-- 存量未收尾的本地上传会话补记为 true，使升级后仍能断点续传。
UPDATE "file_uploads" SET "buffer_local" = TRUE WHERE "source_id" = '';
