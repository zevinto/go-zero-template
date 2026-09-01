// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/infrastructure/redisx"
	"github.com/zevinto/go-zero-template/internal/infrastructure/store"
	"github.com/zevinto/go-zero-template/internal/model/users"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	// 可选依赖：未在配置中启用时为 nil，logic 使用前需判空。
	//
	// 注意：DB 的查询/写入应收敛在 model 层（model 持有连接，
	// 由 NewServiceContext 构造时传入），logic 不直接用 DB 写 SQL——
	// 由下方各 Model 实例承载，业务查询一律走 Model。
	DB          sqlx.SqlConn
	RedisClient *redis.Redis

	// Users 用户数据访问，logic 层经 svcCtx.Users 使用。
	Users users.UsersModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := store.NewStore(c.Database)
	logx.Must(err) // 配置了数据库但连不上 → 启动失败

	rdb, err := redisx.NewRedis(c.Redis)
	logx.Must(err)

	sc := &ServiceContext{
		Config:      c,
		DB:          db,
		RedisClient: rdb,
	}

	// 数据库为可选依赖：未配置时 DB 为 nil，model 也不装配，
	// 逻辑层使用前需判空（svcCtx.Users == nil）。
	if db != nil {
		sc.Users = users.NewUsersModel(db)
	}

	return sc
}
