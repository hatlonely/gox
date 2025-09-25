package aggregation

import "fmt"

// MinAggregation 最小值聚合
type MinAggregation struct {
	MetricAggregation
}

func (a *MinAggregation) Type() AggregationType {
	return AggTypeMin
}

func (a *MinAggregation) ToES() map[string]interface{} {
	return map[string]interface{}{
		"min": map[string]interface{}{
			"field": a.Field,
		},
	}
}

func (a *MinAggregation) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("MIN(%s) AS %s", a.Field, a.AggName), nil, nil
}

func (a *MinAggregation) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"$min": "$" + a.Field,
	}, nil
}