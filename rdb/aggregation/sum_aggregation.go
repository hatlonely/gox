package aggregation

import "fmt"

// SumAggregation 求和聚合
type SumAggregation struct {
	MetricAggregation
}

func (a *SumAggregation) Type() AggregationType {
	return AggTypeSum
}

func (a *SumAggregation) ToES() map[string]interface{} {
	return map[string]interface{}{
		"sum": map[string]interface{}{
			"field": a.Field,
		},
	}
}

func (a *SumAggregation) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("SUM(%s) AS %s", a.Field, a.AggName), nil, nil
}

func (a *SumAggregation) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"$sum": "$" + a.Field,
	}, nil
}