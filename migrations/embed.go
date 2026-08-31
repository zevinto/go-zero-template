package migrations

import "embed"

// FS 包含当前目录下全部 up/down SQL，供独立迁移命令与集成测试使用。
//
//go:embed *.sql
var FS embed.FS
