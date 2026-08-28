package xerror

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrapPreservesCause(t *testing.T) {
	root := fmt.Errorf("connection refused")
	e := Wrap(root, CodeServiceUnavailable, "存储服务不可用")

	if !errors.Is(e, root) {
		t.Fatal("errors.Is(wrapped, root) = false, want true")
	}
	if e.Msg() != "存储服务不可用" || e.Code() != CodeServiceUnavailable {
		t.Fatalf("unexpected wrapped error: %+v", e)
	}
}

func TestErrorString(t *testing.T) {
	if got, want := NewCodeError(CodeNotFound, "资源不存在").Error(), "40400: 资源不存在"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	e := Wrap(fmt.Errorf("timeout"), CodeInternal, "系统繁忙")
	if got, want := e.Error(), "50000: 系统繁忙: timeout"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// 同 Code 的自定义错误应命中哨兵错误，这是 errors.Is 的设计语义。
func TestIsMatchesSameCode(t *testing.T) {
	custom := Validationf("收货地址超出配送范围")

	if !errors.Is(custom, ErrValidation) {
		t.Fatal("errors.Is(custom, ErrValidation) = false, want true for same code")
	}
	if errors.Is(custom, ErrNotFound) {
		t.Fatal("errors.Is(custom, ErrNotFound) = true, want false for different code")
	}
}

// target 本身是包装错误时，Is 也应沿其链找到同码错误。
func TestIsMatchesWrappedTarget(t *testing.T) {
	wrappedTarget := fmt.Errorf("查询用户: %w", ErrNotFound)
	custom := NewCodeError(CodeNotFound, "用户不存在")

	if !errors.Is(custom, wrappedTarget) {
		t.Fatal("errors.Is(custom, wrappedTarget) = false, want true")
	}
}
