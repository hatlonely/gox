package aggregation

// AggregationType 聚合类型
type AggregationType string

const (
	AggTypeSum         AggregationType = "sum"
	AggTypeAvg         AggregationType = "avg"
	AggTypeMax         AggregationType = "max"
	AggTypeMin         AggregationType = "min"
	AggTypeCount       AggregationType = "count"
	AggTypeTerms       AggregationType = "terms"
	AggTypeHistogram   AggregationType = "histogram"
	AggTypeDateHisto   AggregationType = "date_histogram"
	AggTypeComposite   AggregationType = "composite"
)

// Aggregation 聚合接口
type Aggregation interface {
	Type() AggregationType
	Name() string

	// 后端适配器接口
	ToES() map[string]interface{}
	ToSQL() (string, []interface{}, error)
	ToMongo() (map[string]interface{}, error)
}

