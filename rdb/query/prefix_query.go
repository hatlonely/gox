package query

import "fmt"

// PrefixQuery 前缀查询
type PrefixQuery struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func (q *PrefixQuery) Type() QueryType {
	return QueryTypePrefix
}

func (q *PrefixQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"prefix": map[string]interface{}{
			q.Field: q.Value,
		},
	}
}

func (q *PrefixQuery) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("%s LIKE ?", q.Field), []interface{}{q.Value + "%"}, nil
}

func (q *PrefixQuery) ToMongo() (map[string]interface{}, error) {
	// 使用正则表达式进行前缀匹配
	return map[string]interface{}{
		q.Field: map[string]interface{}{
			"$regex": "^" + q.Value,
			"$options": "i",
		},
	}, nil
}