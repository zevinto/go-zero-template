// Package migrate 基于 golang-migrate 封装迁移执行，SQL 从 embed.FS 读取。
// DSN 由 adapters.go 的 SourceURL 按数据库 adapter 生成。
package migrate

import (
	"errors"
	"fmt"
	"io/fs"

	gmmigrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"  // 注册 mysql:// scheme
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // 注册 pgx5:// scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate 应用全部待执行迁移。无待执行迁移（ErrNoChange）视为成功。
func Migrate(fsys fs.FS, dsn string) error {
	m, err := newMigrator(fsys, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, gmmigrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateDown 回滚 steps 个已应用的迁移版本。
func MigrateDown(fsys fs.FS, dsn string, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("rollback steps must be positive")
	}

	m, err := newMigrator(fsys, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil && !errors.Is(err, gmmigrate.ErrNoChange) {
		return fmt.Errorf("rollback migrations: %w", err)
	}
	return nil
}

// MigrateVersion 返回当前 schema 版本与 dirty 标记；未应用任何迁移时返回 (0, false, nil)。
func MigrateVersion(fsys fs.FS, dsn string) (version uint, dirty bool, err error) {
	m, err := newMigrator(fsys, dsn)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if errors.Is(err, gmmigrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read migration version: %w", err)
	}
	return version, dirty, nil
}

// MigrateTo 精确迁移到指定版本（可上可下）。已在目标版本时视为成功（ErrNoChange）。
// dirty 状态下返回 golang-migrate 的 ErrDirty，需先 Force 修复版本号再继续。
func MigrateTo(fsys fs.FS, dsn string, version uint) error {
	m, err := newMigrator(fsys, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && !errors.Is(err, gmmigrate.ErrNoChange) {
		return fmt.Errorf("migrate to %d: %w", version, err)
	}
	return nil
}

// Force 把 schema 版本号强制改写为 version（不含 dirty 标记），用于修复 dirty 状态。
// version 需为目标迁移文件的版本号；传 -1 表示不做任何版本记录（清空版本表条目）。
// 使用前务必确认目标版本与迁移文件一致，否则会导致版本错位。
func Force(fsys fs.FS, dsn string, version int) error {
	m, err := newMigrator(fsys, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version to %d: %w", version, err)
	}
	return nil
}

func newMigrator(fsys fs.FS, dsn string) (*gmmigrate.Migrate, error) {
	source, err := iofs.New(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("init migration source: %w", err)
	}
	m, err := gmmigrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("init migrate: %w", err)
	}
	return m, nil
}
