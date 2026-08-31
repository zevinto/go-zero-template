// Package store 提供数据库的运行时连接工厂。
// 连接实例由 svc 装配持有，logic 层经 ServiceContext 使用，不在业务代码中自建连接。
// Redis 连接工厂见同级的 redisx 包。
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/zevinto/go-zero-template/internal/config"

	_ "github.com/go-sql-driver/mysql" // 注册 mysql 驱动
	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx 驱动（postgres）
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 连接池默认值，DatabaseConf 对应字段为零值时生效。
const (
	defaultMaxOpenConns       = 100
	defaultMaxIdleConns       = 10
	defaultConnMaxLifetimeSec = 1800 // 30 分钟，避免依赖 DB 侧的静默断连回收
	pingTimeout               = 3 * time.Second
)

// NewStore 按配置建立数据库连接：打开驱动、设置连接池、ping 验证连通性，
// 返回 go-zero 的 SqlConn（自带慢查询日志、耗时指标、trace、熔断与 TransactCtx 事务）。
// Database.Adapter 未配置时返回 (nil, nil)——数据库是可选依赖；
// 配置了但连不上则返回错误，由启动方 fail-fast。
func NewStore(c config.DatabaseConf) (sqlx.SqlConn, error) {
	if c.Adapter == "" {
		return nil, nil
	}

	dsn, err := runtimeDSN(c)
	if err != nil {
		return nil, err
	}

	// 不用 sqlx.NewPostgres/NewMysql：那两个只是 NewSqlConn(driverName, dsn) 的糖，
	// 且驱动名固定（postgres 走 lib/pq）。这里保留我们选择的驱动（pgx）与 DSN 拼装，
	// 经 FromDB 包装获得 SqlConn 的日志/指标/trace/熔断/事务能力；需要裸连接时 RawDB() 取回。
	driver := driverPostgres
	if c.Adapter == "mysql" {
		driver = driverMySQL
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(intOrDefault(c.MaxOpenConns, defaultMaxOpenConns))
	db.SetMaxIdleConns(intOrDefault(c.MaxIdleConns, defaultMaxIdleConns))
	db.SetConnMaxLifetime(time.Duration(lifetimeOrDefault(c.ConnMaxLifetimeSec)) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return sqlx.NewSqlConnFromDB(db), nil
}

func intOrDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func lifetimeOrDefault(sec int) int {
	if sec <= 0 {
		return defaultConnMaxLifetimeSec
	}
	return sec
}
