package query

import (
	"fmt"
	"strings"
)

// BoolQuery 布尔查询
type BoolQuery struct {
	Must           []Query `json:"must,omitempty"`
	Should         []Query `json:"should,omitempty"`
	MustNot        []Query `json:"must_not,omitempty"`
	Filter         []Query `json:"filter,omitempty"`
	MinShouldMatch *int    `json:"minimum_should_match,omitempty"`
}

func (q *BoolQuery) Type() QueryType {
	return QueryTypeBool
}

func (q *BoolQuery) ToES() map[string]interface{} {
	result := make(map[string]interface{})
	boolQuery := make(map[string]interface{})

	if len(q.Must) > 0 {
		must := make([]interface{}, len(q.Must))
		for i, query := range q.Must {
			must[i] = query.ToES()
		}
		boolQuery["must"] = must
	}

	if len(q.Should) > 0 {
		should := make([]interface{}, len(q.Should))
		for i, query := range q.Should {
			should[i] = query.ToES()
		}
		boolQuery["should"] = should
	}

	if len(q.MustNot) > 0 {
		mustNot := make([]interface{}, len(q.MustNot))
		for i, query := range q.MustNot {
			mustNot[i] = query.ToES()
		}
		boolQuery["must_not"] = mustNot
	}

	if len(q.Filter) > 0 {
		filter := make([]interface{}, len(q.Filter))
		for i, query := range q.Filter {
			filter[i] = query.ToES()
		}
		boolQuery["filter"] = filter
	}

	if q.MinShouldMatch != nil {
		boolQuery["minimum_should_match"] = *q.MinShouldMatch
	}

	result["bool"] = boolQuery
	return result
}

func (q *BoolQuery) ToSQL() (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	if len(q.Must) > 0 {
		mustConditions := make([]string, 0, len(q.Must))
		for _, query := range q.Must {
			sql, queryArgs, err := query.ToSQL()
			if err != nil {
				return "", nil, err
			}
			mustConditions = append(mustConditions, sql)
			args = append(args, queryArgs...)
		}
		if len(mustConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(mustConditions, " AND ")+")")
		}
	}

	if len(q.Filter) > 0 {
		filterConditions := make([]string, 0, len(q.Filter))
		for _, query := range q.Filter {
			sql, queryArgs, err := query.ToSQL()
			if err != nil {
				return "", nil, err
			}
			filterConditions = append(filterConditions, sql)
			args = append(args, queryArgs...)
		}
		if len(filterConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(filterConditions, " AND ")+")")
		}
	}

	if len(q.Should) > 0 {
		shouldConditions := make([]string, 0, len(q.Should))
		for _, query := range q.Should {
			sql, queryArgs, err := query.ToSQL()
			if err != nil {
				return "", nil, err
			}
			shouldConditions = append(shouldConditions, sql)
			args = append(args, queryArgs...)
		}
		
		if len(shouldConditions) > 0 {
			// 如果设置了 MinShouldMatch 且不为1，使用条件计数方案
			if q.MinShouldMatch != nil && *q.MinShouldMatch != 1 {
				caseConditions := make([]string, len(shouldConditions))
				for i, condition := range shouldConditions {
					caseConditions[i] = fmt.Sprintf("CASE WHEN (%s) THEN 1 ELSE 0 END", condition)
				}
				shouldSQL := fmt.Sprintf("(%s) >= %d", strings.Join(caseConditions, " + "), *q.MinShouldMatch)
				conditions = append(conditions, shouldSQL)
			} else {
				// 默认行为：使用 OR 连接
				conditions = append(conditions, "("+strings.Join(shouldConditions, " OR ")+")")
			}
		}
	}

	if len(q.MustNot) > 0 {
		mustNotConditions := make([]string, 0, len(q.MustNot))
		for _, query := range q.MustNot {
			sql, queryArgs, err := query.ToSQL()
			if err != nil {
				return "", nil, err
			}
			mustNotConditions = append(mustNotConditions, "NOT ("+sql+")")
			args = append(args, queryArgs...)
		}
		if len(mustNotConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(mustNotConditions, " AND ")+")")
		}
	}

	if len(conditions) == 0 {
		return "1=1", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}

func (q *BoolQuery) ToMongo() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	andConditions := make([]interface{}, 0)

	if len(q.Must) > 0 {
		for _, query := range q.Must {
			condition, err := query.ToMongo()
			if err != nil {
				return nil, err
			}
			andConditions = append(andConditions, condition)
		}
	}

	if len(q.Filter) > 0 {
		for _, query := range q.Filter {
			condition, err := query.ToMongo()
			if err != nil {
				return nil, err
			}
			andConditions = append(andConditions, condition)
		}
	}

	if len(q.Should) > 0 {
		orConditions := make([]interface{}, 0, len(q.Should))
		for _, query := range q.Should {
			condition, err := query.ToMongo()
			if err != nil {
				return nil, err
			}
			orConditions = append(orConditions, condition)
		}
		
		// 如果设置了 MinShouldMatch 且不为1，使用 $expr 条件计数方案
		if q.MinShouldMatch != nil && *q.MinShouldMatch != 1 {
			condArray := make([]interface{}, len(orConditions))
			for i, condition := range orConditions {
				condArray[i] = map[string]interface{}{
					"$cond": []interface{}{condition, 1, 0},
				}
			}
			
			exprCondition := map[string]interface{}{
				"$expr": map[string]interface{}{
					"$gte": []interface{}{
						map[string]interface{}{
							"$add": condArray,
						},
						*q.MinShouldMatch,
					},
				},
			}
			andConditions = append(andConditions, exprCondition)
		} else {
			// 默认行为：使用 $or
			andConditions = append(andConditions, map[string]interface{}{"$or": orConditions})
		}
	}

	if len(q.MustNot) > 0 {
		norConditions := make([]interface{}, 0, len(q.MustNot))
		for _, query := range q.MustNot {
			condition, err := query.ToMongo()
			if err != nil {
				return nil, err
			}
			norConditions = append(norConditions, condition)
		}
		andConditions = append(andConditions, map[string]interface{}{"$nor": norConditions})
	}

	if len(andConditions) == 0 {
		return map[string]interface{}{}, nil
	}

	if len(andConditions) == 1 {
		if mongoCondition, ok := andConditions[0].(map[string]interface{}); ok {
			return mongoCondition, nil
		}
	}

	result["$and"] = andConditions
	return result, nil
}