// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package health

import (
	"context"

	"github.com/zevinto/go-zero-template/internal/svc"
	"github.com/zevinto/go-zero-template/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 健康检查
func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingLogic) Ping(req *types.PingRequest) (resp *types.PingResponse, err error) {
	return &types.PingResponse{
		Message: "pong",
	}, nil
}
