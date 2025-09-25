package aggregation

import "fmt"

// MaxAggregation 最大值聚合
type MaxAggregation struct {
	MetricAggregation
}

func (a *MaxAggregation) Type() AggregationType {
	return AggTypeMax
}

func (a *MaxAggregation) ToES() map[string]interface{} {
	return map[string]interface{}{
		"max": map[string]interface{}{
			"field": a.Field,
		},
	}
}

func (a *MaxAggregation) ToSQL() (string, []interface{}, error) {
	return fmt.Sprintf("MAX(%s) AS %s", a.Field, a.AggName), nil, nil
}

func (a *MaxAggregation) ToMongo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"$max": "$" + a.Field,
	}, nil
}