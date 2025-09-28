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
	// 对于MongoDB，count聚合通常就是简单的计数
	// 如果指定了字段，可以使用$sum配合条件，但为简化起见，我们使用基本计数
	return map[string]interface{}{
		"$sum": 1,
	}, nil
}