package query

import "fmt"

// RegexpQuery 正则表达式查询
type RegexpQuery struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func (q *RegexpQuery) Type() QueryType {
	return QueryTypeRegexp
}

func (q *RegexpQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"regexp": map[string]interface{}{
			q.Field: q.Value,
		},
	}
}

func (q *RegexpQuery) ToSQL() (string, []interface{}, error) {
	// MySQL使用REGEXP，PostgreSQL使用~操作符
	// 这里使用通用的REGEXP关键字
	return fmt.Sprintf("%s REGEXP ?", q.Field), []interface{}{q.Value}, nil
}

func (q *RegexpQuery) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		q.Field: map[string]interface{}{
			"$regex": q.Value,
			"$options": "i",
		},
	}, nil
}