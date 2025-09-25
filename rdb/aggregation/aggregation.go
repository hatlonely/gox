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

// MetricAggregation 指标聚合基础结构
type MetricAggregation struct {
	AggName string
	Field   string
}

func (m *MetricAggregation) Name() string {
	return m.AggName
}

// BucketAggregation 桶聚合基础结构
type BucketAggregation struct {
	AggName         string
	Field           string
	SubAggregations []Aggregation
}

func (b *BucketAggregation) Name() string {
	return b.AggName
}

// 构建子聚合的通用方法
func buildSubAggregations(subAggs []Aggregation) map[string]interface{} {
	if len(subAggs) == 0 {
		return nil
	}
	
	aggs := make(map[string]interface{})
	for _, subAgg := range subAggs {
		aggs[subAgg.Name()] = subAgg.ToES()
	}
	return aggs
}

func buildSubAggregationsSQL(subAggs []Aggregation) ([]string, []interface{}, error) {
	if len(subAggs) == 0 {
		return nil, nil, nil
	}
	
	var sqls []string
	var args []interface{}
	
	for _, subAgg := range subAggs {
		sql, subArgs, err := subAgg.ToSQL()
		if err != nil {
			return nil, nil, err
		}
		sqls = append(sqls, sql)
		args = append(args, subArgs...)
	}
	
	return sqls, args, nil
}

func buildSubAggregationsMongo(subAggs []Aggregation) (map[string]interface{}, error) {
	if len(subAggs) == 0 {
		return nil, nil
	}
	
	pipeline := make(map[string]interface{})
	for _, subAgg := range subAggs {
		subResult, err := subAgg.ToMongo()
		if err != nil {
			return nil, err
		}
		pipeline[subAgg.Name()] = subResult
	}
	
	return pipeline, nil
}