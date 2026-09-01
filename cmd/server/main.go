// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/joho/godotenv"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/handler"
	"github.com/zevinto/go-zero-template/internal/infrastructure/apollo"
	"github.com/zevinto/go-zero-template/internal/infrastructure/migrate"
	"github.com/zevinto/go-zero-template/internal/svc"
	"github.com/zevinto/go-zero-template/internal/xresponse"
	"github.com/zevinto/go-zero-template/internal/xvalidator"
	"github.com/zevinto/go-zero-template/migrations"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/server.yaml", "the config file")

func main() {
	flag.Parse()
	_ = godotenv.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	// 配置中心（冷加载）：非 dev 拉取失败直接启动失败；dev 降级为本地 yaml。
	// Overlay 在 map 层合并后一次性加载，本地必填字段（Name 等）不受影响。
	if c.Apollo.MetaAddr != "" {
		remote, err := apollo.Fetch(context.Background(), c.Apollo)
		switch {
		case err != nil && c.Mode == service.DevMode:
			logx.Errorf("拉取配置中心失败，降级为本地配置: %v", err)
		case err != nil:
			logx.Must(err)
		default:
			logx.Must(apollo.Overlay(*configFile, remote, &c))
		}
	}

	httpx.SetValidator(xvalidator.New())
	xresponse.Setup()

	// 单实例内部系统：可选在启动时自动应用数据库迁移（配置 MigrateOnStart: true）。
	// 需同时配置 Database 连接；迁移失败则启动失败（fail-fast），避免带错误 schema 起服务。
	if c.MigrateOnStart {
		if c.Database.Adapter == "" {
			logx.Must(fmt.Errorf("MigrateOnStart=true 但未配置 Database.Adapter（可选: postgres, mysql）"))
		}
		dsn, err := migrate.SourceURL(c.Database)
		logx.Must(err)
		logx.Must(migrate.Migrate(migrations.FS, dsn))
		logx.Info("数据库迁移已应用到最新版本")
	}

	server := rest.MustNewServer(c.RestConf)

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	logx.Infof("Starting server at %s:%d...", c.Host, c.Port)
	server.Start()
}
