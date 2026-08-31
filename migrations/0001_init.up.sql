-- 示例迁移：演示命名约定与基本表结构，正式项目请替换为真实 schema。
-- 命名约定：NNNN_描述.up.sql / NNNN_描述.down.sql，序号四位递增，成对出现。
CREATE TABLE IF NOT EXISTS users (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
