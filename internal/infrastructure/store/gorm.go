// gorm 连接工厂：提供 gorm（*gorm.DB）作为数据访问的 ORM 连接。
// 复用 runtimeDSN 拼接连接串；未配置 Database.Adapter 返回 (nil, nil)（可选依赖）。
// 迁移路径仍走 golang-migrate（见 migrate 包），不启用 gorm.AutoMigrate。
//
// gorm 日志统一接到 go-zero 的 logx：级别由 Database.GormLogLevel 控制
// （silent/error/warn/info，空取 warn——只输出慢 SQL 与错误）。
package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zevinto/go-zero-template/internal/config"
)

// 慢 SQL 阈值，超过则走 logx.Slow 级别输出。
const gormSlowThreshold = 200 * time.Millisecond

// NewGorm 建立 gorm 连接：拼接 DSN、打开驱动、设置连接池、接 logx 日志。
func NewGorm(c config.DatabaseConf) (*gorm.DB, error) {
	if c.Adapter == "" {
		return nil, nil
	}

	dsn, err := runtimeDSN(c)
	if err != nil {
		return nil, err
	}

	var dialector gorm.Dialector
	switch c.Adapter {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的 Database.Adapter %q（可选: postgres, mysql）", c.Adapter)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logxGormLogger(gormLogLevel(c.GormLogLevel)),
	})
	if err != nil {
		return nil, err
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(intOrDefault(c.MaxOpenConns, defaultMaxOpenConns))
		sqlDB.SetMaxIdleConns(intOrDefault(c.MaxIdleConns, defaultMaxIdleConns))
		sqlDB.SetConnMaxLifetime(time.Duration(lifetimeOrDefault(c.ConnMaxLifetimeSec)) * time.Second)
	}

	return db, nil
}

// gormLogLevel 解析配置级别；空默认 warn。
func gormLogLevel(s string) glogger.LogLevel {
	switch s {
	case "silent":
		return glogger.Silent
	case "error":
		return glogger.Error
	case "warn":
		return glogger.Warn
	case "info":
		return glogger.Info
	default:
		return glogger.Warn
	}
}

// logxLogger 将 gorm 的日志重定向到 go-zero 的 logx，
// 使 SQL/慢查询/错误与项目统一日志体系（trace、采集）一致。
type logxLogger struct {
	glogger.Interface
	level glogger.LogLevel
}

// logxGormLogger 返回接入 go-zero logx 的 gorm logger。
func logxGormLogger(level glogger.LogLevel) glogger.Interface {
	return &logxLogger{level: level}
}

func (l *logxLogger) LogMode(level glogger.LogLevel) glogger.Interface {
	return &logxLogger{level: level}
}

func (l *logxLogger) Info(_ context.Context, format string, args ...any) {
	if l.level >= glogger.Info {
		logx.Infof(format, args...)
	}
}

func (l *logxLogger) Warn(_ context.Context, format string, args ...any) {
	if l.level >= glogger.Warn {
		logx.Slowf(format, args...)
	}
}

func (l *logxLogger) Error(_ context.Context, format string, args ...any) {
	if l.level >= glogger.Error {
		logx.Errorf(format, args...)
	}
}

// Trace 在每次 SQL 执行时被调用：err 非空记错误，慢于阈值记慢日志，
// 否则在 Info 级别时记普通 SQL。
func (l *logxLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()

	switch {
	case err != nil && l.level >= glogger.Error:
		logx.WithContext(ctx).Errorf("gorm sql err: %s | %s", err, sql)
	case elapsed > gormSlowThreshold && l.level >= glogger.Warn:
		logx.WithContext(ctx).Slowf("gorm slow sql (%s): %s", elapsed, sql)
	case l.level >= glogger.Info:
		logx.WithContext(ctx).Infof("gorm sql (%s): %s", elapsed, sql)
	}
}
