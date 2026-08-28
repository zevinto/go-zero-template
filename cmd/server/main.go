// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/handler"
	"github.com/zevinto/go-zero-template/internal/svc"
	"github.com/zevinto/go-zero-template/internal/xresponse"
	"github.com/zevinto/go-zero-template/internal/xvalidator"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/server.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	httpx.SetValidator(xvalidator.New())
	xresponse.Setup()

	server := rest.MustNewServer(c.RestConf)

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	logx.Infof("Starting server at %s:%d...", c.Host, c.Port)
	server.Start()
}
