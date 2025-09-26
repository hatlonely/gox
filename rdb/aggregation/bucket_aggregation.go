package aggregation

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