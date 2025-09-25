package query

import (
	"fmt"
	"strings"
)

// RangeQuery 范围查询
type RangeQuery struct {
	Field string                 `json:"field"`
	Gt    interface{}            `json:"gt,omitempty"`
	Gte   interface{}            `json:"gte,omitempty"`
	Lt    interface{}            `json:"lt,omitempty"`
	Lte   interface{}            `json:"lte,omitempty"`
	Extra map[string]interface{} `json:"extra,omitempty"`
}

func (q *RangeQuery) Type() QueryType {
	return QueryTypeRange
}

func (q *RangeQuery) ToES() map[string]interface{} {
	rangeQuery := make(map[string]interface{})
	
	if q.Gt != nil {
		rangeQuery["gt"] = q.Gt
	}
	if q.Gte != nil {
		rangeQuery["gte"] = q.Gte
	}
	if q.Lt != nil {
		rangeQuery["lt"] = q.Lt
	}
	if q.Lte != nil {
		rangeQuery["lte"] = q.Lte
	}
	
	// 添加额外字段
	for k, v := range q.Extra {
		rangeQuery[k] = v
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			q.Field: rangeQuery,
		},
	}
}

func (q *RangeQuery) ToSQL() (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	if q.Gt != nil {
		conditions = append(conditions, fmt.Sprintf("%s > ?", q.Field))
		args = append(args, q.Gt)
	}
	if q.Gte != nil {
		conditions = append(conditions, fmt.Sprintf("%s >= ?", q.Field))
		args = append(args, q.Gte)
	}
	if q.Lt != nil {
		conditions = append(conditions, fmt.Sprintf("%s < ?", q.Field))
		args = append(args, q.Lt)
	}
	if q.Lte != nil {
		conditions = append(conditions, fmt.Sprintf("%s <= ?", q.Field))
		args = append(args, q.Lte)
	}

	if len(conditions) == 0 {
		return "1=1", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}

func (q *RangeQuery) ToMongo() (map[string]interface{}, error) {
	condition := make(map[string]interface{})

	if q.Gt != nil {
		condition["$gt"] = q.Gt
	}
	if q.Gte != nil {
		condition["$gte"] = q.Gte
	}
	if q.Lt != nil {
		condition["$lt"] = q.Lt
	}
	if q.Lte != nil {
		condition["$lte"] = q.Lte
	}

	return map[string]interface{}{
		q.Field: condition,
	}, nil
}