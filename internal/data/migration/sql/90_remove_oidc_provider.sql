-- Momoko 不再内置 OIDC 身份提供方（Momoko 账号的 OIDC Provider），清理它留下的数据：
--   1. 数据表 oidc_clients（ent 自动迁移不会删除已不在 schema 中的表；授权码只存在内存里，没有表）；
--   2. configs 表中的 oidc.* 配置键（启用状态、Issuer、各令牌有效期、签名私钥与轮换时间）；
--   3. operation_logs 表中 OIDC 操作类型的记录（这些操作类型已删除）。
-- 「查看 OIDC」「编辑 OIDC」两个内置按钮菜单不在这里删：启动时的默认菜单同步会删除已不在内置列表里的内置菜单，
-- 角色对它们的授权（role_menus）随菜单外键级联删除。
-- 标识符使用双引号，MySQL 下由执行器改写为反引号。

DROP TABLE IF EXISTS "oidc_clients";

DELETE FROM "configs" WHERE "key" LIKE 'oidc.%';

DELETE FROM "operation_logs" WHERE "operation_type" LIKE 'OperationTypeOIDC%';
