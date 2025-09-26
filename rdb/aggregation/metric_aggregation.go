package aggregation

// MetricAggregation 指标聚合基础结构
type MetricAggregation struct {
	AggName string
	Field   string
}

func (m *MetricAggregation) Name() string {
	return m.AggName
}