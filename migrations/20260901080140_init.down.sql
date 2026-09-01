-- +migrate Down

-- 与 up 成对的回滚：删除 init 迁移创建的表。
DROP TABLE IF EXISTS users;