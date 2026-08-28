package health

import (
	"context"
	"testing"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/svc"
	"github.com/zevinto/go-zero-template/internal/types"
)

// 业务逻辑层单测：直接构造 Logic，不经过 HTTP。
func TestPing(t *testing.T) {
	l := NewPingLogic(context.Background(), svc.NewServiceContext(config.Config{}))

	resp, err := l.Ping(&types.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
	if resp == nil || resp.Message != "pong" {
		t.Fatalf("Ping() resp = %+v, want message \"pong\"", resp)
	}
}
