package query

import "fmt"

// MatchQuery 全文搜索查询
type MatchQuery struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

func (q *MatchQuery) Type() QueryType {
	return QueryTypeMatch
}

func (q *MatchQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"match": map[string]interface{}{
			q.Field: q.Value,
		},
	}
}

func (q *MatchQuery) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("%s LIKE ?", q.Field), []interface{}{"%" + fmt.Sprintf("%v", q.Value) + "%"}, nil
}

func (q *MatchQuery) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		q.Field: map[string]interface{}{
			"$regex": q.Value,
			"$options": "i",
		},
	}, nil
}