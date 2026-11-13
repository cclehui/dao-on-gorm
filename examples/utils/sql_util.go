package utils

import "strings"

// SQLUtilType SQL 相关的纯函数集合.
//
// 和 base_utils 的 IntUtil / StrUtil 一致: 无状态, 定义成类型 + 全局实例,
// 调用方写 SQLUtil.EscapeLike(s), 后续再加方法不用改调用点.
type SQLUtilType struct{}

// SQLUtil 全局实例.
var SQLUtil SQLUtilType

// EscapeLike 转义 LIKE 模式里的通配符, 让用户输入按字面量匹配.
//
// 不转义的话 "a_b" 会匹配 "axb", 单独的 "%" 会匹配所有行.
// 反斜杠必须最先替换, 否则会把后面新增的转义符再转义一遍.
// 转义符固定为反斜杠, SQL 里要同时写 escape '\', 否则 MySQL 默认转义符不生效.
func (u SQLUtilType) EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)

	return s
}

// LikeContains 把用户输入包装成"包含"匹配的模式, 输入部分已转义.
func (u SQLUtilType) LikeContains(s string) string {
	return "%" + u.EscapeLike(s) + "%"
}
