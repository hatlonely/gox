package query

import "fmt"

// ExistsQuery 字段存在查询
type ExistsQuery struct {
	Field string `json:"field"`
}

func (q *ExistsQuery) Type() QueryType {
	return QueryTypeExists
}

func (q *ExistsQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"exists": map[string]interface{}{
			"field": q.Field,
		},
	}
}

func (q *ExistsQuery) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("%s IS NOT NULL", q.Field), nil, nil
}

func (q *ExistsQuery) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		q.Field: map[string]interface{}{
			"$exists": true,
		},
	}, nil
}