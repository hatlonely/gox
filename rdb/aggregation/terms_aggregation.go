package aggregation

import (
	"fmt"
	"strings"
)

// TermsAggregation 词条聚合
type TermsAggregation struct {
	BucketAggregation
	Size  int
	Order map[string]string // 排序字段和方向
}

func (a *TermsAggregation) Type() AggregationType {
	return AggTypeTerms
}

func (a *TermsAggregation) ToES() map[string]interface{} {
	terms := map[string]interface{}{
		"field": a.Field,
	}
	
	if a.Size > 0 {
		terms["size"] = a.Size
	}
	
	if len(a.Order) > 0 {
		terms["order"] = a.Order
	}
	
	result := map[string]interface{}{
		"terms": terms,
	}
	
	if subAggs := buildSubAggregations(a.SubAggregations); subAggs != nil {
		result["aggs"] = subAggs
	}
	
	return result
}

func (a *TermsAggregation) ToSQL() (string, []interface{}, error) {
	var parts []string
	var args []interface{}
	
	groupBy := fmt.Sprintf("GROUP BY %s", a.Field)
	parts = append(parts, groupBy)
	
	if subSQLs, subArgs, err := buildSubAggregationsSQL(a.SubAggregations); err != nil {
		return "", nil, err
	} else if len(subSQLs) > 0 {
		parts = append(parts, strings.Join(subSQLs, ", "))
		args = append(args, subArgs...)
	}
	
	if len(a.Order) > 0 {
		var orderParts []string
		for field, direction := range a.Order {
			orderParts = append(orderParts, fmt.Sprintf("%s %s", field, strings.ToUpper(direction)))
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}
	
	if a.Size > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", a.Size))
	}
	
	return strings.Join(parts, " "), args, nil
}

func (a *TermsAggregation) ToMongo() (map[string]interface{}, error) {
	groupStage := map[string]interface{}{
		"$group": map[string]interface{}{
			"_id": "$" + a.Field,
		},
	}
	
	if subAggs, err := buildSubAggregationsMongo(a.SubAggregations); err != nil {
		return nil, err
	} else if subAggs != nil {
		for name, agg := range subAggs {
			groupStage["$group"].(map[string]interface{})[name] = agg
		}
	}
	
	pipeline := []interface{}{groupStage}
	
	if len(a.Order) > 0 {
		sortStage := map[string]interface{}{"$sort": a.Order}
		pipeline = append(pipeline, sortStage)
	}
	
	if a.Size > 0 {
		limitStage := map[string]interface{}{"$limit": a.Size}
		pipeline = append(pipeline, limitStage)
	}
	
	return map[string]interface{}{
		"$facet": map[string]interface{}{
			a.AggName: pipeline,
		},
	}, nil
}