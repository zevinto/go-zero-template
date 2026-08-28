// Package xvalidator 包装 go-playground/validator 并挂载到 go-zero 的
// httpx.SetValidator，对 HTTP 请求参数做声明式校验。
//
// 校验规则以 validate 标签声明在 .api 文件的类型字段上：goctl 会把手写的
// validate 标签原样生成到 internal/types（goctl 1.10.2 已验证），
// 因此规则随代码生成存活，无需自定义 goctl 模板。
//
// 校验失败返回 *xerror.CodeError（CodeBadRequest），由 xresponse 统一转成
// 40000 + 可读文案；文案描述的是调用方自己的请求参数，可以安全透出。
package xvalidator

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/zevinto/go-zero-template/internal/xerror"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// v 是线程安全的，进程内全局复用。
var v = validator.New(validator.WithRequiredStructEnabled())

// New 返回挂载到 httpx.SetValidator 的全局校验器，
// 在 main 中调用 httpx.SetValidator(New()) 完成接线。
func New() httpx.Validator {
	return validatorFunc(validate)
}

// validatorFunc 适配器：go-zero 未提供函数到 httpx.Validator 的转换，自行封装。
type validatorFunc func(*http.Request, any) error

func (f validatorFunc) Validate(r *http.Request, data any) error {
	return f(r, data)
}

// validate 校验请求 struct 的 validate 标签，失败归一为 CodeBadRequest。
func validate(_ *http.Request, data any) error {
	err := v.Struct(data)
	if err == nil {
		return nil
	}

	// 非 struct 输入（InvalidValidationError）无法校验，跳过不拦截。
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}

	return xerror.NewCodeError(xerror.CodeBadRequest, formatIssues(ve))
}

// formatIssues 将字段级校验失败转为可读文案，如：
// "参数校验失败: Email 邮箱格式不正确, Password 不能为空"。
func formatIssues(ve validator.ValidationErrors) string {
	var b strings.Builder
	b.WriteString("参数校验失败")
	for i, fe := range ve {
		sep := ":"
		if i > 0 {
			sep = ","
		}
		fmt.Fprintf(&b, "%s %s %s", sep, fe.Field(), tagDesc(fe.Tag(), fe.Param()))
	}
	return b.String()
}

// tagDesc 将规则标签转为可读描述，param 为规则参数（如 min=6 中的 "6"）；
// 未收录的标签降级为展示标签名。
func tagDesc(tag, param string) string {
	switch tag {
	case "required":
		return "不能为空"
	case "email":
		return "邮箱格式不正确"
	case "url":
		return "URL 格式不正确"
	case "oneof":
		return fmt.Sprintf("可选值: %s", strings.ReplaceAll(param, " ", " / "))
	case "len":
		return paramDesc(param, "长度必须为 %s", tag)
	case "min":
		return paramDesc(param, "长度/大小不能少于 %s", tag)
	case "max":
		return paramDesc(param, "长度/大小不能超过 %s", tag)
	case "gte":
		return paramDesc(param, "不能小于 %s", tag)
	case "lte":
		return paramDesc(param, "不能大于 %s", tag)
	case "gt":
		return paramDesc(param, "必须大于 %s", tag)
	case "lt":
		return paramDesc(param, "必须小于 %s", tag)
	default:
		if param != "" {
			return fmt.Sprintf("不满足 %s=%s 规则", tag, param)
		}
		return fmt.Sprintf("不满足 %s 规则", tag)
	}
}

// paramDesc 带参数的描述；param 缺失时降级为展示标签名，避免出现残缺文案。
func paramDesc(param, format, tag string) string {
	if param == "" {
		return fmt.Sprintf("不满足 %s 规则", tag)
	}
	return fmt.Sprintf(format, param)
}
