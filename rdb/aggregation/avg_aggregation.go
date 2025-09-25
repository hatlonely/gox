package aggregation

import "fmt"

// AvgAggregation 平均值聚合
type AvgAggregation struct {
	MetricAggregation
}

func (a *AvgAggregation) Type() AggregationType {
	return AggTypeAvg
}

func (a *AvgAggregation) ToES() map[string]interface{} {
	return map[string]interface{}{
		"avg": map[string]interface{}{
			"field": a.Field,
		},
	}
}

func (a *AvgAggregation) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("AVG(%s) AS %s", a.Field, a.AggName), nil, nil
}

func (a *AvgAggregation) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"$avg": "$" + a.Field,
	}, nil
}