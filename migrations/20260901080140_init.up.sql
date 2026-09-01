-- +migrate Up

-- 初始化迁移：创建基础用户表。
-- 命名采用时间戳 YYYYMMDDHHMMSS，后续迁移用更大时间戳保证顺序递增。
CREATE TABLE IF NOT EXISTS users (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       VARCHAR(64) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +migrate Down
