package xresponse

import (
	"context"
	"errors"
	"testing"

	"github.com/zevinto/go-zero-template/internal/xerror"
)

func TestFromCodeError(t *testing.T) {
	resp := errEnvelope(xerror.NewCodeErrorf(xerror.CodeValidation, "字段 %s 不合法", "age"))

	if resp.Code != int(xerror.CodeValidation) {
		t.Fatalf("code = %d, want %d", resp.Code, int(xerror.CodeValidation))
	}
	if resp.Message != "字段 age 不合法" {
		t.Fatalf("message = %q, want %q", resp.Message, "字段 age 不合法")
	}
	if resp.Data == nil {
		t.Fatal("data = nil, want empty object")
	}
}

func TestBodyConvertsTypedCode(t *testing.T) {
	resp := Body(xerror.CodeConflict, "冲突", map[string]int{"retryAfter": 5})

	if resp.Code != int(xerror.CodeConflict) {
		t.Fatalf("code = %d, want %d", resp.Code, int(xerror.CodeConflict))
	}
}

func TestOkEnvelope(t *testing.T) {
	resp := okEnvelope(map[string]string{"message": "pong"})

	if resp.Code != int(xerror.CodeSuccess) || resp.Message != "success" {
		t.Fatalf("unexpected envelope: %+v", resp)
	}
	if resp.Data == nil {
		t.Fatal("data = nil, want business payload")
	}
}

func TestErrorResponseBusinessError(t *testing.T) {
	status, body := errorResponse(context.Background(), xerror.ErrUnauthorized)

	resp, ok := body.(*Response)
	if !ok {
		t.Fatalf("body type = %T, want *Response", body)
	}
	if status != 200 || resp.Code != int(xerror.CodeUnauthorized) || resp.Message != "账号未登录" {
		t.Fatalf("unexpected response: status=%d body=%+v", status, resp)
	}
}

// go-zero 未导出解析错误类型，参数错误按文案特征识别后归入 40000。
func TestErrorResponseParseError(t *testing.T) {
	// 真实 go-zero mapping 错误文案：field "id" is not set
	parseErr := errors.New(`field "id" is not set`)

	status, body := errorResponse(context.Background(), parseErr)
	resp := body.(*Response)

	if status != 200 || resp.Code != int(xerror.CodeBadRequest) {
		t.Fatalf("unexpected response: status=%d body=%+v", status, resp)
	}
	if resp.Message != "请求参数错误" {
		t.Fatalf("message = %q, want fixed message", resp.Message)
	}
	if resp.Data == nil {
		t.Fatal("data = nil, want empty object")
	}
}

func TestErrorResponseInternalErrorNoLeak(t *testing.T) {
	secret := errors.New("dial tcp 10.0.0.8:3306: connection refused; password=secret")

	status, body := errorResponse(context.Background(), secret)
	resp := body.(*Response)

	if status != 200 || resp.Code != int(xerror.CodeInternal) {
		t.Fatalf("unexpected response: status=%d body=%+v", status, resp)
	}
	if resp.Message != "系统繁忙，请稍后重试" {
		t.Fatalf("message = %q, want fixed message", resp.Message)
	}
	if resp.Data == nil {
		t.Fatal("data = nil, want empty object")
	}
}
