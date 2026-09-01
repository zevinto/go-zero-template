// 数据库迁移命令：读取与 server 相同的配置文件，执行 migrations/ 下的版本化迁移。
//
// 基于 spf13/cobra 实现多子命令与详细帮助：运行 migrate --help 查看全部命令，
// 每个子命令可独立查看 migrate <cmd> --help。
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/infrastructure/apollo"
	"github.com/zevinto/go-zero-template/internal/infrastructure/migrate"
	"github.com/zevinto/go-zero-template/internal/xutil"
	"github.com/zevinto/go-zero-template/migrations"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

// configFile 由 root 的 persistent flag --file/-f 注入，子命令共享。
var configFile string

func main() {
	rootCmd := &cobra.Command{
		Use:   "migrate [command]",
		Short: "数据库版本化迁移工具",
		Long: xutil.Dedent(`读取与 server 相同的配置文件，执行 migrations/ 下的 golang-migrate 版本化迁移。

		用法: migrate [command] [flags]

		命令:
		  migrate up            应用全部待执行迁移
		  migrate down          回滚 N 个版本（默认 1）
		  migrate <version>     精确迁移到指定版本（可上可下）
		  migrate version       查看当前版本与 dirty 状态
		  migrate force         修复 dirty 状态，强制改写版本号`),
	}
	rootCmd.PersistentFlags().StringVarP(&configFile, "file", "f", "etc/server.yaml",
		"配置文件路径（默认 etc/server.yaml）")

	rootCmd.AddCommand(
		newUpCmd(),
		newDownCmd(),
		newMigrateToCmd(),
		newVersionCmd(),
		newForceCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		// cobra 已打印错误；此处仅返回非零退出码。
		os.Exit(1)
	}
}

// runWithDSN 先做统一的配置加载（.env → yaml → Apollo 覆盖 → DSN），
// 再执行 fn。fn 接收 cobra 本轮参数 args 与计算出的 dsn，子命令可从中取需要的部分。
func runWithDSN(fn func(args []string, dsn string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_ = godotenv.Load()

		var c config.Config
		conf.MustLoad(configFile, &c, conf.UseEnv())

		if c.Apollo.MetaAddr != "" {
			remote, err := apollo.Fetch(context.Background(), c.Apollo)
			switch {
			case err != nil && c.Mode == service.DevMode:
				logx.Errorf("拉取配置中心失败，降级为本地配置: %v", err)
			case err != nil:
				return fmt.Errorf("拉取配置中心失败: %w", err)
			default:
				if err := apollo.Overlay(configFile, remote, &c); err != nil {
					return err
				}
			}
		}

		if c.Database.Adapter == "" {
			return fmt.Errorf("配置缺少 Database.Adapter（可选: postgres, mysql），见 etc/server.yaml 中 Database 段示例")
		}
		dsn, err := migrate.SourceURL(c.Database)
		if err != nil {
			return fmt.Errorf("构造连接串失败: %w", err)
		}
		return fn(args, dsn)
	}
}

// newUpCmd 应用全部待执行迁移。
func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "应用全部待执行迁移",
		Long: xutil.Dedent(`应用 migrations/ 下所有尚未执行的迁移（沿 up 版本递增）。
			无待执行迁移时视为成功，不报错。`),
		Example: "  migrate -f etc/server.yaml up",
		Args:    cobra.NoArgs,
		RunE: runWithDSN(func(args []string, dsn string) error {
			if err := migrate.Migrate(migrations.FS, dsn); err != nil {
				return fmt.Errorf("迁移失败: %w", err)
			}
			fmt.Println("迁移执行完成")
			return nil
		}),
	}
}

// newDownCmd 回滚 N 个版本。
func newDownCmd() *cobra.Command {
	var steps int
	cmd := &cobra.Command{
		Use:   "down",
		Short: "回滚 N 个版本（默认 1）",
		Long:  "沿 down 版本回滚 --steps 指定的版本数（默认 1），steps 必须是正整数。",
		Example: "  migrate -f etc/server.yaml down\n" +
			"  migrate -f etc/server.yaml down --steps 3",
		Args: cobra.NoArgs,
		RunE: runWithDSN(func(args []string, dsn string) error {
			if err := migrate.MigrateDown(migrations.FS, dsn, steps); err != nil {
				return fmt.Errorf("回滚失败: %w", err)
			}
			fmt.Printf("已回滚 %d 个版本\n", steps)
			return nil
		}),
	}
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "回滚的版本数（正整数）")
	return cmd
}

// newMigrateToCmd 精确迁移到指定版本。
func newMigrateToCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "migrate <version>",
		Short:   "精确迁移到指定版本（可上可下）",
		Long:    "精确迁移（Migrate）到指定版本号：目标大于当前则向上，小于则向下回滚。适合跨版本分阶段升级。",
		Example: "  migrate -f etc/server.yaml migrate 3",
		Args:    cobra.ExactArgs(1),
		RunE: runWithDSN(func(args []string, dsn string) error {
			target, err := positiveVersion(args[0])
			if err != nil {
				return err
			}
			if err := migrate.MigrateTo(migrations.FS, dsn, target); err != nil {
				return fmt.Errorf("迁移失败: %w", err)
			}
			fmt.Printf("已迁移到版本 %d\n", target)
			return nil
		}),
	}
}

// newVersionCmd 查看当前版本与 dirty 状态。
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "查看当前 schema 版本与 dirty 状态",
		Long:    "输出当前已应用的最高迁移版本号与 dirty 标记。未应用任何迁移时显示提示。",
		Example: "  migrate -f etc/server.yaml version",
		Args:    cobra.NoArgs,
		RunE: runWithDSN(func(args []string, dsn string) error {
			version, dirty, err := migrate.MigrateVersion(migrations.FS, dsn)
			if err != nil {
				return fmt.Errorf("读取版本失败: %w", err)
			}
			if version == 0 {
				fmt.Println("当前版本: 未应用任何迁移")
				return nil
			}
			fmt.Printf("当前版本: %d (dirty: %v)\n", version, dirty)
			return nil
		}),
	}
}

// newForceCmd 修复 dirty 状态，强制改写版本号。
func newForceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "force <version>",
		Short: "修复 dirty 状态，强制改写版本号",
		Long: xutil.Dedent(`把 schema 版本号强制改写为指定值（不含 dirty 标记），用于修复迁移中途失败留下的 dirty 状态。
			version 为迁移文件的版本号；传 -1 表示清空版本记录（不指向任何迁移）。
			使用前务必确认目标版本号与数据库实际迁移到的文件一致，否则会导致版本错位。`),
		Example: "  migrate -f etc/server.yaml force 3\n" +
			"  migrate -f etc/server.yaml force -1",
		Args: cobra.ExactArgs(1),
		RunE: runWithDSN(func(args []string, dsn string) error {
			v, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("force 需要版本号参数（整数，或 -1 清空版本记录），收到: %q", args[0])
			}
			if v < -1 {
				return fmt.Errorf("force 版本号不能小于 -1，收到: %d", v)
			}
			if err := migrate.Force(migrations.FS, dsn, v); err != nil {
				return fmt.Errorf("force 失败: %w", err)
			}
			fmt.Printf("已将 schema 版本强制改写为 %d (dirty 已清除)\n", v)
			return nil
		}),
	}
}

// positiveVersion 解析"必须为正"的版本号参数（migrate <version> 用）。
func positiveVersion(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v == 0 {
		return 0, fmt.Errorf("需要正整数的迁移版本号，收到 %q", s)
	}
	return uint(v), nil
}
