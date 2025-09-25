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