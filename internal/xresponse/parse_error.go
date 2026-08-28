// 参数解析错误的识别逻辑。
//
// 本文件是与 go-zero 版本强耦合的部分：go-zero 未导出可判定的解析错误
// 类型（errors.As 接不到），只能按错误文案特征识别。升级 go-zero 后若
// core/mapping 或 json 解析的错误文案变化，仅需对照维护本文件。
package xresponse

import "strings"

// parseErrorMarkers 是 go-zero 参数解析错误的文案特征片段，覆盖 core/mapping
// （字段未填、类型不匹配、range/options 违规等）与 encoding/json（语法、类型、截断）两类。
var parseErrorMarkers = [...]string{
	"is not set",                 // field %q is not set
	"is not fully set",           // %q is not fully set
	"type mismatch for field",    // 类型不匹配
	"unmarshal field",            // 解析字段失败
	"is not settable",            // 字段不可设置
	"is not defined in options",  // 超出可选值范围
	"wrong number range setting", // 超出 range 范围
	"mustn't be nil",             // 必填字段为 nil
	"invalid character",          // json 语法错误
	"cannot unmarshal",           // json 类型错误
	"unexpected EOF",             // json 截断
}

// isRequestParseError 判断 err 是否为请求参数解析类错误（客户端问题）。
// 根治方案是在自定义 goctl handler 模板中将解析错误包装为
// xerror.BadRequestf（typed），届时本文件可整体删除。
func isRequestParseError(err error) bool {
	msg := err.Error()
	for _, m := range parseErrorMarkers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}
