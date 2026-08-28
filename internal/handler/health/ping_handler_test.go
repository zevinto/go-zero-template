package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/svc"
	"github.com/zevinto/go-zero-template/internal/xresponse"
)

// HTTP 层集成测试：走真实 handler 与统一响应包装，验证最终响应体。
func TestPingHandler(t *testing.T) {
	xresponse.Setup()

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rec := httptest.NewRecorder()
	PingHandler(svc.NewServiceContext(config.Config{}))(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", rec.Body.String(), err)
	}
	if body.Code != 0 || body.Message != "success" || body.Data.Message != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

// 端到端验证参数解析错误的兜底：走真实 httpx.Parse 失败路径，
// 断言归入 40000 且不透传原始请求内容。
func TestPingHandlerParseError(t *testing.T) {
	xresponse.Setup()

	req := httptest.NewRequest(http.MethodPost, "/api/ping", strings.NewReader(`{"bad json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	PingHandler(svc.NewServiceContext(config.Config{}))(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "bad json") {
		t.Fatalf("raw request leaked into response: %s", rec.Body.String())
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", rec.Body.String(), err)
	}
	if body.Code != 40000 || body.Message != "请求参数错误" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
