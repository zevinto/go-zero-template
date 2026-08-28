// Package xresponse 统一响应包装：所有接口的响应都为 {code, message, data} 结构。
// 包装通过 Setup 安装到 go-zero 的 httpx 上，handler/logic 保持生成器原生写法即可。
package xresponse

import (
	"context"
	"errors"
	"net/http"

	"github.com/zevinto/go-zero-template/internal/xerror"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Response 所有接口的统一响应体。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Body 构造指定 code/message/data 的响应体。
func Body(code xerror.Code, message string, data any) *Response {
	return &Response{
		Code:    int(code),
		Message: message,
		Data:    data,
	}
}

// okEnvelope 成功响应，data 为业务数据。
func okEnvelope(data any) *Response {
	return Body(xerror.CodeSuccess, "success", data)
}

// errEnvelope 将业务错误转成响应体，data 统一为空对象。
func errEnvelope(e *xerror.CodeError) *Response {
	return Body(e.Code(), e.Msg(), map[string]any{})
}

// errorResponse 是所有错误的统一出口：业务错误直接转换；
// 参数解析类错误按客户端错误返回；其余按内部错误兜底。
// 两条兜底路径都只对外给固定文案，原始信息只进日志。
func errorResponse(ctx context.Context, err error) (int, any) {
	var bizErr *xerror.CodeError
	if errors.As(err, &bizErr) {
		return http.StatusOK, errEnvelope(bizErr)
	}

	if isRequestParseError(err) {
		// 客户端问题不打 Error 级别避免误告警，但保留明细便于定位。
		logx.WithContext(ctx).Infof("bad request: %v", err)
		return http.StatusOK, Body(xerror.CodeBadRequest, "请求参数错误", map[string]any{})
	}

	logx.WithContext(ctx).Errorf("unhandled error: %+v", err)
	return http.StatusOK, errEnvelope(xerror.ErrInternal)
}

// Setup 安装成功/错误的统一包装逻辑，必须在服务启动前调用一次。
//
// 约定：HTTP 状态码一律返回 200，客户端以响应体中的 code 判断结果；
// 若有网关等场景需要还原真实 HTTP 状态码，调整 errorResponse 中各分支的第一个返回值即可。
func Setup() {
	httpx.SetOkHandler(func(_ context.Context, v any) any {
		return okEnvelope(v)
	})

	httpx.SetErrorHandlerCtx(errorResponse)
}
