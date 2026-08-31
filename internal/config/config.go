// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	// Database 数据库连接，迁移命令与运行时共用；不配置则数据库为可选依赖。
	// 通过 Adapter 切换数据库类型，字段含义各数据库通用。
	Database DatabaseConf `json:",optional"`

	// Redis 可选；未配置 Host 时 ServiceContext 中为 nil。
	Redis redis.RedisConf `json:",optional"`
}

type DatabaseConf struct {
	// Adapter 数据库类型：postgres / mysql（迁移驱动按此切换）
	Adapter  string `json:",optional"`
	Host     string `json:",optional"`
	Port     int    `json:",optional"` // 0 表示按 Adapter 取默认端口（postgres 5432 / mysql 3306）
	Username string `json:",optional"`
	Password string `json:",optional"`
	Name     string `json:",optional"` // 数据库名

	// Params 附加连接参数（sslmode、charset 等），按 adapter 语义追加到连接串。
	Params map[string]string `json:",optional"`

	// 连接池参数，零值时由 store 取默认值（100 / 10 / 1800 秒）。
	MaxOpenConns       int `json:",optional"`
	MaxIdleConns       int `json:",optional"`
	ConnMaxLifetimeSec int `json:",optional"`
}
