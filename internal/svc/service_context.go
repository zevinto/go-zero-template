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
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config

	// DB 为 go-zero 原生（sqlx）连接，保留以承载非 gorm 的原生查询/既有逻辑。
	// 可选依赖：未在配置中启用时为 nil，logic 使用前需判空。
	DB          sqlx.SqlConn
	RedisClient *redis.Redis

	// Gorm 为 gorm 连接，model 层（如 users）经此做 ORM 数据访问。
	Gorm *gorm.DB

	// Users 用户数据访问（gorm 实现），logic 层经 svcCtx.Users 使用。
	Users users.Users
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := store.NewStore(c.Database)
	logx.Must(err) // 配置了数据库但连不上 → 启动失败

	rdb, err := redisx.NewRedis(c.Redis)
	logx.Must(err)

	gdb, err := store.NewGorm(c.Database)
	logx.Must(err)

	sc := &ServiceContext{
		Config:      c,
		DB:          db,
		RedisClient: rdb,
		Gorm:        gdb,
	}

	// 数据库为可选依赖：未配置时 Gorm 为 nil，model 也不装配，
	// 逻辑层使用前需判空（svcCtx.Users == nil）。
	if gdb != nil {
		sc.Users = users.NewUsersModel(gdb)
	}

	return sc
}
