package query

import (
	"fmt"
	"strings"
)

// WildcardQuery 通配符查询
type WildcardQuery struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func (q *WildcardQuery) Type() QueryType {
	return QueryTypeWildcard
}

func (q *WildcardQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"wildcard": map[string]interface{}{
			q.Field: q.Value,
		},
	}
}

func (q *WildcardQuery) ToSQL() (string, []interface{}, error) {
	// 将通配符转换为SQL LIKE模式
	// * -> %（匹配任意数量字符）
	// ? -> _（匹配单个字符）
	pattern := strings.ReplaceAll(q.Value, "*", "%")
	pattern = strings.ReplaceAll(pattern, "?", "_")
	return fmt.Sprintf("%s LIKE ?", q.Field), []interface{}{pattern}, nil
}

func (q *WildcardQuery) ToMongo() (map[string]interface{}, error) {
	// 将通配符转换为正则表达式
	// * -> .*（匹配任意数量字符）
	// ? -> .（匹配单个字符）
	pattern := strings.ReplaceAll(q.Value, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	// 转义其他特殊字符
	pattern = "^" + pattern + "$"
	
	return map[string]interface{}{
		q.Field: map[string]interface{}{
			"$regex": pattern,
			"$options": "i",
		},
	}, nil
}