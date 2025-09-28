package aggregation

import "fmt"

// CountAggregation 计数聚合
type CountAggregation struct {
	MetricAggregation
}

func (a *CountAggregation) Type() AggregationType {
	return AggTypeCount
}

func (a *CountAggregation) ToES() map[string]interface{} {
	return map[string]interface{}{
		"value_count": map[string]interface{}{
			"field": a.Field,
		},
	}
}

func (a *CountAggregation) ToSQL() (string, []interface{}, error) {
	if a.Field == "" {
		return fmt.Sprintf("COUNT(*) AS %s", a.AggName), nil, nil
	}
	return fmt.Sprintf("COUNT(%s) AS %s", a.Field, a.AggName), nil, nil
}

func (a *CountAggregation) ToMongo() (map[string]interface{}, error) {
	if a.Field == "" {
		// 简单文档计数 - COUNT(*)
		return map[string]interface{}{
			"$sum": 1,
		}, nil
	}
	
	// 字段非空计数 - COUNT(field)
	// 统计字段存在且不为null、不为空字符串的文档数量
	return map[string]interface{}{
		"$sum": map[string]interface{}{
			"$cond": map[string]interface{}{
				"if": map[string]interface{}{
					"$and": []interface{}{
						// 字段不为null
						map[string]interface{}{"$ne": []interface{}{"$" + a.Field, nil}},
						// 字段存在（不是missing）
						map[string]interface{}{"$ne": []interface{}{"$" + a.Field, "$$REMOVE"}},
						// 如果是字符串类型，还要检查不为空字符串
						map[string]interface{}{
							"$cond": map[string]interface{}{
								"if":   map[string]interface{}{"$eq": []interface{}{map[string]interface{}{"$type": "$" + a.Field}, "string"}},
								"then": map[string]interface{}{"$ne": []interface{}{"$" + a.Field, ""}},
								"else": true,
							},
						},
					},
				},
				"then": 1,
				"else": 0,
			},
		},
	}, nil
}