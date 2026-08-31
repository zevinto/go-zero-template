// 数据库迁移命令：读取与 server 相同的配置文件，执行 migrations/ 下的版本化迁移。
//
// 用法：
//
//	migrate -f etc/server.yaml up            # 应用全部待执行迁移
//	migrate -f etc/server.yaml down -steps 1 # 回滚 N 个版本
//	migrate -f etc/server.yaml version       # 查看当前版本与 dirty 状态
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/infrastructure/migrate"
	"github.com/zevinto/go-zero-template/migrations"

	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	configFile := flag.String("f", "etc/server.yaml", "the config file")
	steps := flag.Int("steps", 1, "steps for down")
	flag.Parse()

	command := flag.Arg(0)
	if command == "" {
		fatal("用法: migrate [-f 配置文件] up|down|version [-steps N]")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if c.Database.Adapter == "" {
		fatal("配置缺少 Database.Adapter（可选: postgres, mysql），见 etc/server.yaml 中 Database 段示例")
	}
	dsn, err := migrate.SourceURL(c.Database)
	if err != nil {
		fatal("构造连接串失败: %v", err)
	}

	switch command {
	case "up":
		if err := migrate.Migrate(migrations.FS, dsn); err != nil {
			fatal("迁移失败: %v", err)
		}
		fmt.Println("迁移执行完成")
	case "down":
		if err := migrate.MigrateDown(migrations.FS, dsn, *steps); err != nil {
			fatal("回滚失败: %v", err)
		}
		fmt.Printf("已回滚 %d 个版本\n", *steps)
	case "version":
		version, dirty, err := migrate.MigrateVersion(migrations.FS, dsn)
		if err != nil {
			fatal("读取版本失败: %v", err)
		}
		if version == 0 {
			fmt.Println("当前版本: 未应用任何迁移")
			return
		}
		fmt.Printf("当前版本: %d (dirty: %v)\n", version, dirty)
	default:
		fatal("未知命令 %q，可用命令: up | down | version", command)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
