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
		return map[string]interface{}{
			"$sum": 1,
		}, nil
	}
	return map[string]interface{}{
		"$sum": map[string]interface{}{
			"$cond": []interface{}{
				map[string]interface{}{"$ne": []interface{}{"$" + a.Field, nil}},
				1,
				0,
			},
		},
	}, nil
}