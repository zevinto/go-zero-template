// Package xutil 提供跨命令/包复用的通用工具函数。
// 命名沿用 x* 系列约定（xerror/xresponse/xvalidator）：与 go-zero 既有包区分、
// 非 go-zero 原生、可被任意包引用。
package xutil

import "strings"

// Dedent 去掉多行字符串的公共前导缩进与首尾空行。
// 常用于 cobra 等 CLI 的多行帮助文本：源码里用统一缩进对齐美观，
// 运行时剥掉公共缩进让输出整齐顶格；每行的相对缩进保留。
func Dedent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}

	// 公共前导缩进 = 从第二行起，所有非空行里最短的前导空白序列。
	// 不计第一行：它紧跟反引号、与源码首列对齐，是段落开头，不参与公共缩进。
	var prefix string
	start := 1
	if len(lines) == 1 {
		start = 0
	}
	for _, ln := range lines[start:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		p := leadingWhitespace(ln)
		if prefix == "" || len(p) < len(prefix) {
			prefix = p
		}
		if prefix == "" {
			break // 有一行使缩进归零，公共缩进即空
		}
	}

	out := make([]string, len(lines))
	for i, ln := range lines {
		if ln == "" {
			out[i] = ""
			continue
		}
		out[i] = strings.TrimPrefix(ln, prefix)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// leadingWhitespace 返回行首的空白前缀（空格或 tab）。
func leadingWhitespace(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return line[:i]
		}
	}
	return line
}
