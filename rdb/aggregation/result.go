package aggregation

// AggregationResult 聚合结果接口
type AggregationResult interface {
	// Get 获取聚合结果
	Get(aggName string) interface{}
	
	// GetBuckets 获取桶聚合结果
	GetBuckets(aggName string) []Bucket
	
	// GetValue 获取指标聚合的数值结果
	GetValue(aggName string) float64
	
	// GetCount 获取文档计数
	GetCount(aggName string) int64
}

// Bucket 桶结果接口
type Bucket interface {
	// Key 桶的键值
	Key() interface{}
	
	// DocCount 文档数量
	DocCount() int64
	
	// SubAggregations 子聚合结果
	SubAggregations() AggregationResult
}

// DefaultAggregationResult 默认聚合结果实现
type DefaultAggregationResult struct {
	results map[string]interface{}
}

func NewAggregationResult() *DefaultAggregationResult {
	return &DefaultAggregationResult{
		results: make(map[string]interface{}),
	}
}

func (r *DefaultAggregationResult) SetResult(aggName string, value interface{}) {
	r.results[aggName] = value
}

func (r *DefaultAggregationResult) Get(aggName string) interface{} {
	return r.results[aggName]
}

func (r *DefaultAggregationResult) GetBuckets(aggName string) []Bucket {
	if buckets, ok := r.results[aggName].([]Bucket); ok {
		return buckets
	}
	return nil
}

func (r *DefaultAggregationResult) GetValue(aggName string) float64 {
	if value, ok := r.results[aggName].(float64); ok {
		return value
	}
	if value, ok := r.results[aggName].(int64); ok {
		return float64(value)
	}
	if value, ok := r.results[aggName].(int); ok {
		return float64(value)
	}
	return 0.0
}

func (r *DefaultAggregationResult) GetCount(aggName string) int64 {
	if value, ok := r.results[aggName].(int64); ok {
		return value
	}
	if value, ok := r.results[aggName].(int); ok {
		return int64(value)
	}
	return 0
}

// DefaultBucket 默认桶实现
type DefaultBucket struct {
	key             interface{}
	docCount        int64
	subAggregations AggregationResult
}

func NewBucket(key interface{}, docCount int64) *DefaultBucket {
	return &DefaultBucket{
		key:             key,
		docCount:        docCount,
		subAggregations: NewAggregationResult(),
	}
}

func (b *DefaultBucket) Key() interface{} {
	return b.key
}

func (b *DefaultBucket) DocCount() int64 {
	return b.docCount
}

func (b *DefaultBucket) SubAggregations() AggregationResult {
	return b.subAggregations
}

func (b *DefaultBucket) SetSubAggregation(aggName string, value interface{}) {
	if defaultResult, ok := b.subAggregations.(*DefaultAggregationResult); ok {
		defaultResult.SetResult(aggName, value)
	}
}