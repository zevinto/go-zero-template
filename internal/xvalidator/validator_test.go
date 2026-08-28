package xvalidator

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zevinto/go-zero-template/internal/xerror"
	"github.com/zevinto/go-zero-template/internal/xresponse"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type registerReq struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=6"`
}

func TestValidatePasses(t *testing.T) {
	req := &registerReq{Email: "a@b.com", Password: "123456"}

	if err := validate(httptest.NewRequest(http.MethodGet, "/", nil), req); err != nil {
		t.Fatalf("validate() = %v, want nil", err)
	}
}

// 校验失败应归一为 CodeBadRequest，且文案包含全部字段问题。
func TestValidateReturnsBadRequest(t *testing.T) {
	req := &registerReq{Email: "not-an-email", Password: "123"}

	err := validate(httptest.NewRequest(http.MethodGet, "/", nil), req)

	var ce *xerror.CodeError
	if !errors.As(err, &ce) {
		t.Fatalf("error type = %T, want *xerror.CodeError", err)
	}
	if ce.Code() != xerror.CodeBadRequest {
		t.Fatalf("code = %d, want %d", ce.Code(), int(xerror.CodeBadRequest))
	}
	msg := ce.Msg()
	for _, want := range []string{"参数校验失败", "Email", "邮箱格式不正确", "Password", "长度/大小不能少于 6"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}
}

// 常用规则标签的中文描述，含参数拼接与缺失降级。
func TestTagDesc(t *testing.T) {
	cases := []struct {
		tag   string
		param string
		want  string
	}{
		{"required", "", "不能为空"},
		{"email", "", "邮箱格式不正确"},
		{"url", "", "URL 格式不正确"},
		{"min", "6", "长度/大小不能少于 6"},
		{"max", "32", "长度/大小不能超过 32"},
		{"gte", "0", "不能小于 0"},
		{"oneof", "1 2 3", "可选值: 1 / 2 / 3"},
		{"startswith", "usr_", "不满足 startswith=usr_ 规则"}, // 未收录标签带参数
		{"unknownrule", "", "不满足 unknownrule 规则"},        // 未收录标签无参数
		{"min", "", "不满足 min 规则"},                        // 参数缺失时降级
	}
	for _, c := range cases {
		if got := tagDesc(c.tag, c.param); got != c.want {
			t.Errorf("tagDesc(%q, %q) = %q, want %q", c.tag, c.param, got, c.want)
		}
	}
}

// 非 struct 输入（如空请求的零值）不应被拦截。
func TestValidateSkipsNonStruct(t *testing.T) {
	var n int
	if err := validate(httptest.NewRequest(http.MethodGet, "/", nil), n); err != nil {
		t.Fatalf("validate(non-struct) = %v, want nil", err)
	}
}

// 端到端：注册为全局校验器后，httpx.Parse 应在校验失败时得到 CodeError。
// 注意请求需带上全部 form 参数：go-zero 的默认必填检查（不带 optional 的
// form 标签）先于 validator 执行，缺参数时会在 Parse 阶段就被拦截，
// 走不到 validator 钩子——两层校验的正确分工是 go-zero 管存在性、
// validator 管格式与复杂规则。
func TestParseTriggersValidator(t *testing.T) {
	httpx.SetValidator(New())

	r := httptest.NewRequest(http.MethodGet, "/register?email=bad&password=123456", nil)
	var req registerReq

	err := httpx.Parse(r, &req)

	var ce *xerror.CodeError
	if !errors.As(err, &ce) || ce.Code() != xerror.CodeBadRequest {
		t.Fatalf("httpx.Parse error = %v, want CodeError(%d)", err, int(xerror.CodeBadRequest))
	}
}

// 全链路：validator 错误经 handler 的 httpx.ErrorCtx 与 xresponse 包装，
// 最终响应体应为统一格式 {code:40000, message:"参数校验失败: ...", data:{}}。
func TestValidatorErrorThroughResponseEnvelope(t *testing.T) {
	httpx.SetValidator(New())
	xresponse.Setup()

	r := httptest.NewRequest(http.MethodGet, "/register?email=bad&password=123456", nil)
	var req registerReq
	if err := httpx.Parse(r, &req); err == nil {
		t.Fatal("httpx.Parse error = nil, want validation error")
	}

	rec := httptest.NewRecorder()
	httpx.ErrorCtx(r.Context(), rec, validate(nil, &req))

	var body struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", rec.Body.String(), err)
	}
	if body.Code != int(xerror.CodeBadRequest) {
		t.Fatalf("code = %d, want %d", body.Code, int(xerror.CodeBadRequest))
	}
	if !strings.Contains(body.Message, "参数校验失败") || !strings.Contains(body.Message, "Email") {
		t.Fatalf("message = %q, want validation details", body.Message)
	}
	if body.Data == nil {
		t.Fatal("data = nil, want empty object")
	}
}
