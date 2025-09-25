package query

import "fmt"

// TermQuery 精确匹配查询
type TermQuery struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

func (q *TermQuery) Type() QueryType {
	return QueryTypeTerm
}

func (q *TermQuery) ToES() map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{
			q.Field: q.Value,
		},
	}
}

func (q *TermQuery) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("%s = ?", q.Field), []interface{}{q.Value}, nil
}

func (q *TermQuery) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		q.Field: q.Value,
	}, nil
}