// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"

	"github.com/zevinto/go-zero-template/internal/infrastructure/apollo"
)

type Config struct {
	rest.RestConf

	// Database 数据库连接，迁移命令与运行时共用；不配置则数据库为可选依赖。
	// 通过 Adapter 切换数据库类型，字段含义各数据库通用。
	//
	// tag 约定：key 一律用字段名（`json:",optional"`），不要自定小写/下划线 key——
	// go-zero conf 匹配时对 key 做小写化，大小写随意，但下划线不参与归一，
	// 自定 snake_case 会让 yaml 里按字段名写的配置静默失配。
	Database DatabaseConf `json:",optional"`

	// Redis 可选；未配置 Host 时 ServiceContext 中为 nil。
	Redis redis.RedisConf `json:",optional"`

	// Apollo 配置中心引导段：MetaAddr 未配置表示不接入（本地 yaml 为准）。
	// 接入后非 dev 环境启动时拉取并覆盖加载；dev 拉取失败降级为本地 yaml。
	Apollo apollo.Conf `json:",optional"`

	// MigrateOnStart 是否在服务启动时自动应用数据库迁移（单实例内部系统适用）。
	// 默认 false：推荐在部署流水线显式迁移后再启动服务，避免多实例并发迁移；
	// 单实例或开发环境可置 true，随服务启动自动 up 到最新。
	MigrateOnStart bool `json:",optional"`
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

	// GormLogLevel gorm SQL 日志级别：silent / error / warn / info。
	// 空取 warn（只输出慢 SQL 与错误，推荐生产）；调试 SQL 可设 info。
	// 日志统一由 go-zero 的 logx 承接（见 store/gorm.go）。
	GormLogLevel string `json:",optional"`
}
