-- 用户通行证不再提供面向玩家的 OIDC Provider（游戏客户端改用通行证启动票据），清理它留下的数据：
--   1. 数据表 passport_oidc_grants、passport_oidc_clients（ent 自动迁移不会删除已不在 schema 中的表）；
--   2. configs 表中通行证 OIDC 的签名私钥 passport.oidc_private_key；
--   3. 通行证设置 passport.settings（v1.PassportSettings 的 protojson）中已删除的 oidcEnabled 字段，
--      否则读取设置时按未知字段解析失败、退回默认值；protojson 单行输出时逗号后可能带一个空格，两种写法都去掉；
--   4. 游戏账号绑定的绑定方式 oidc 改名为 create（经通行证新建的游戏账号）。
-- 标识符使用双引号，MySQL 下由执行器改写为反引号。

DROP TABLE IF EXISTS "passport_oidc_grants";
DROP TABLE IF EXISTS "passport_oidc_clients";

DELETE FROM "configs" WHERE "key" = 'passport.oidc_private_key';

UPDATE "configs" SET "value" = REPLACE(REPLACE("value", ', "oidcEnabled":true', ''), ',"oidcEnabled":true', '') WHERE "key" = 'passport.settings';

UPDATE "game_account_links" SET "method" = 'create' WHERE "method" = 'oidc';
