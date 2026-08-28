// Package xerror 定义业务错误码与错误类型。
// 业务代码在 logic 层构造 CodeError 返回，由 xresponse 包统一转换为 HTTP 响应。
//
// 错误码分段规则见下方 const 注释；新错误码必须集中在本文件登记，
// 禁止在使用处硬编码魔法数字。
package xerror

import (
	"errors"
	"fmt"
)

// Code 是稳定业务错误码。
type Code int

const (
	CodeSuccess Code = 0 // 成功

	// 客户端类 40000–49999：前三位镜像 HTTP 类别，末两位为该类别下的顺序号。
	// 认证/权限/限流等技术性错误走这里，便于网关与监控归类。
	CodeBadRequest         Code = 40000 // 请求格式或参数解析错误
	CodeUnauthorized       Code = 40100 // 未登录
	CodeInvalidCredentials Code = 40101 // 账号或密码错误
	CodeForbidden          Code = 40300 // 无权限执行该操作
	CodeAccountLocked      Code = 40301 // 账号已锁定
	CodePasswordMustChange Code = 40302 // 需要先修改密码
	CodeNotFound           Code = 40400 // 资源不存在
	CodeConflict           Code = 40900 // 操作冲突、重复提交
	CodeValidation         Code = 42200 // 业务校验失败
	CodeRateLimited        Code = 42900 // 触发限流

	// 服务端类 50000–59999：对外只透出固定文案，详细信息进日志。
	CodeInternal           Code = 50000 // 服务端内部错误
	CodeServiceUnavailable Code = 50300 // 下游依赖不可用

	// 业务模块专属段 10000–39999：领域性错误放这里，每个模块占用一整千位区间，
	// 新模块先在此登记区间再定义错误码。
	// 例：10000–10999 用户模块、11000–11999 订单模块。
)

// 预定义固定错误：Code 与对外文案均已固定，业务侧直接返回。
var (
	ErrBadRequest         = NewCodeError(CodeBadRequest, "请求参数错误")
	ErrUnauthorized       = NewCodeError(CodeUnauthorized, "账号未登录")
	ErrForbidden          = NewCodeError(CodeForbidden, "无权限执行该操作")
	ErrNotFound           = NewCodeError(CodeNotFound, "访问的资源不存在或已删除")
	ErrConflict           = NewCodeError(CodeConflict, "操作冲突，请勿重复提交")
	ErrValidation         = NewCodeError(CodeValidation, "业务校验不通过")
	ErrRateLimited        = NewCodeError(CodeRateLimited, "请求过于频繁，请稍后再试")
	ErrInternal           = NewCodeError(CodeInternal, "系统繁忙，请稍后重试")
	ErrServiceUnavailable = NewCodeError(CodeServiceUnavailable, "服务暂不可用，请稍后重试")
)

// CodeError 业务错误，实现 error 接口。字段非导出，防止外部改动全局哨兵错误的值。
type CodeError struct {
	code  Code
	msg   string
	cause error
}

func NewCodeError(code Code, msg string) *CodeError {
	return &CodeError{code: code, msg: msg}
}

func NewCodeErrorf(code Code, format string, args ...any) *CodeError {
	return NewCodeError(code, fmt.Sprintf(format, args...))
}

// Wrap 用业务语义包装底层错误：对外展示 msg，cause 通过日志排查，
// 并支持上层继续用 errors.Is / errors.As 判断根因。
func Wrap(cause error, code Code, msg string) *CodeError {
	return &CodeError{code: code, msg: msg, cause: cause}
}

// Unwrap 暴露底层原因，便于 errors.Is / errors.As 链式判断。
func (e *CodeError) Unwrap() error { return e.cause }

// Is 支持 errors.Is：同 Code 视为同一类错误
// （如自定义的 Validationf 错误可命中哨兵 ErrValidation）。
// 用 errors.As 而非类型断言，使 target 自身也是包装错误时仍能命中。
func (e *CodeError) Is(target error) bool {
	var t *CodeError
	return errors.As(target, &t) && e.code == t.code
}

// BadRequestf 构造请求参数类错误。
func BadRequestf(format string, args ...any) *CodeError {
	return NewCodeErrorf(CodeBadRequest, format, args...)
}

// Validationf 构造业务校验类错误。
func Validationf(format string, args ...any) *CodeError {
	return NewCodeErrorf(CodeValidation, format, args...)
}

// Error 实现 error 接口，携带完整信息用于日志输出。
func (e *CodeError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%d: %s: %v", e.code, e.msg, e.cause)
	}
	return fmt.Sprintf("%d: %s", e.code, e.msg)
}

// Code 返回业务错误码。
func (e *CodeError) Code() Code { return e.code }

// Msg 返回对外的错误文案。
func (e *CodeError) Msg() string { return e.msg }
