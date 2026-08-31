// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/infrastructure/cache"
	"github.com/zevinto/go-zero-template/internal/infrastructure/store"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	// 可选依赖：未在配置中启用时为 nil，logic 使用前需判空。
	//
	// 注意：DB 的查询/写入应收敛在 model 层（model 持有连接，
	// 由 NewServiceContext 构造时传入），logic 不直接用 DB 写 SQL；
	// 待 model 接管后此字段转为非导出或移除。
	DB          sqlx.SqlConn
	RedisClient *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := store.NewStore(c.Database)
	logx.Must(err) // 配置了数据库但连不上 → 启动失败

	rdb, err := cache.NewRedis(c.Redis)
	logx.Must(err)

	return &ServiceContext{
		Config:      c,
		DB:          db,
		RedisClient: rdb,
	}
}
