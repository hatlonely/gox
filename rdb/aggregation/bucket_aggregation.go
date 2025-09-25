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

// DateHistogramAggregation 日期直方图聚合
type DateHistogramAggregation struct {
	BucketAggregation
	Interval string // "1d", "1M", "1y" 等
	Format   string
	TimeZone string
}

func (a *DateHistogramAggregation) Type() AggregationType {
	return AggTypeDateHisto
}

func (a *DateHistogramAggregation) ToES() map[string]interface{} {
	dateHisto := map[string]interface{}{
		"field":    a.Field,
		"interval": a.Interval,
	}
	
	if a.Format != "" {
		dateHisto["format"] = a.Format
	}
	
	if a.TimeZone != "" {
		dateHisto["time_zone"] = a.TimeZone
	}
	
	result := map[string]interface{}{
		"date_histogram": dateHisto,
	}
	
	if subAggs := buildSubAggregations(a.SubAggregations); subAggs != nil {
		result["aggs"] = subAggs
	}
	
	return result
}

func (a *DateHistogramAggregation) ToSQL() (string, []interface{}, error) {
	var dateFormat string
	
	switch a.Interval {
	case "1d", "day":
		dateFormat = "DATE(%s)"
	case "1M", "month":
		dateFormat = "DATE_FORMAT(%s, '%%Y-%%m-01')"
	case "1y", "year":
		dateFormat = "DATE_FORMAT(%s, '%%Y-01-01')"
	case "1h", "hour":
		dateFormat = "DATE_FORMAT(%s, '%%Y-%%m-%%d %%H:00:00')"
	default:
		dateFormat = "DATE(%s)"
	}
	
	groupBy := fmt.Sprintf("GROUP BY %s", fmt.Sprintf(dateFormat, a.Field))
	
	var parts []string
	var args []interface{}
	
	parts = append(parts, groupBy)
	
	if subSQLs, subArgs, err := buildSubAggregationsSQL(a.SubAggregations); err != nil {
		return "", nil, err
	} else if len(subSQLs) > 0 {
		parts = append(parts, strings.Join(subSQLs, ", "))
		args = append(args, subArgs...)
	}
	
	return strings.Join(parts, " "), args, nil
}

func (a *DateHistogramAggregation) ToMongo() (map[string]interface{}, error) {
	var dateExpression interface{}
	
	switch a.Interval {
	case "1d", "day":
		dateExpression = map[string]interface{}{
			"$dateToString": map[string]interface{}{
				"format": "%Y-%m-%d",
				"date":   "$" + a.Field,
			},
		}
	case "1M", "month":
		dateExpression = map[string]interface{}{
			"$dateToString": map[string]interface{}{
				"format": "%Y-%m",
				"date":   "$" + a.Field,
			},
		}
	case "1y", "year":
		dateExpression = map[string]interface{}{
			"$dateToString": map[string]interface{}{
				"format": "%Y",
				"date":   "$" + a.Field,
			},
		}
	default:
		dateExpression = "$" + a.Field
	}
	
	groupStage := map[string]interface{}{
		"$group": map[string]interface{}{
			"_id": dateExpression,
		},
	}
	
	if subAggs, err := buildSubAggregationsMongo(a.SubAggregations); err != nil {
		return nil, err
	} else if subAggs != nil {
		for name, agg := range subAggs {
			groupStage["$group"].(map[string]interface{})[name] = agg
		}
	}
	
	pipeline := []interface{}{
		groupStage,
		map[string]interface{}{"$sort": map[string]interface{}{"_id": 1}},
	}
	
	return map[string]interface{}{
		"$facet": map[string]interface{}{
			a.AggName: pipeline,
		},
	}, nil
}

// CompositeSource 复合聚合源
type CompositeSource struct {
	Name  string
	Field string
	Order string // "asc" or "desc"
}

// CompositeAggregation 复合聚合
type CompositeAggregation struct {
	BucketAggregation
	Sources []CompositeSource
	Size    int
	After   map[string]interface{}
}

func (a *CompositeAggregation) Type() AggregationType {
	return AggTypeComposite
}

func (a *CompositeAggregation) ToES() map[string]interface{} {
	sources := make([]map[string]interface{}, len(a.Sources))
	for i, source := range a.Sources {
		sourceMap := map[string]interface{}{
			source.Name: map[string]interface{}{
				"terms": map[string]interface{}{
					"field": source.Field,
				},
			},
		}
		
		if source.Order != "" {
			sourceMap[source.Name].(map[string]interface{})["terms"].(map[string]interface{})["order"] = source.Order
		}
		
		sources[i] = sourceMap
	}
	
	composite := map[string]interface{}{
		"sources": sources,
	}
	
	if a.Size > 0 {
		composite["size"] = a.Size
	}
	
	if a.After != nil {
		composite["after"] = a.After
	}
	
	result := map[string]interface{}{
		"composite": composite,
	}
	
	if subAggs := buildSubAggregations(a.SubAggregations); subAggs != nil {
		result["aggs"] = subAggs
	}
	
	return result
}

func (a *CompositeAggregation) ToSQL() (string, []interface{}, error) {
	var groupFields []string
	for _, source := range a.Sources {
		groupFields = append(groupFields, source.Field)
	}
	
	groupBy := fmt.Sprintf("GROUP BY %s", strings.Join(groupFields, ", "))
	
	var parts []string
	var args []interface{}
	
	parts = append(parts, groupBy)
	
	if subSQLs, subArgs, err := buildSubAggregationsSQL(a.SubAggregations); err != nil {
		return "", nil, err
	} else if len(subSQLs) > 0 {
		parts = append(parts, strings.Join(subSQLs, ", "))
		args = append(args, subArgs...)
	}
	
	var orderParts []string
	for _, source := range a.Sources {
		if source.Order != "" {
			orderParts = append(orderParts, fmt.Sprintf("%s %s", source.Field, strings.ToUpper(source.Order)))
		}
	}
	if len(orderParts) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}
	
	if a.Size > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", a.Size))
	}
	
	return strings.Join(parts, " "), args, nil
}

func (a *CompositeAggregation) ToMongo() (map[string]interface{}, error) {
	groupId := make(map[string]interface{})
	for _, source := range a.Sources {
		groupId[source.Name] = "$" + source.Field
	}
	
	groupStage := map[string]interface{}{
		"$group": map[string]interface{}{
			"_id": groupId,
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
	
	sortFields := make(map[string]interface{})
	for _, source := range a.Sources {
		if source.Order == "desc" {
			sortFields["_id."+source.Name] = -1
		} else {
			sortFields["_id."+source.Name] = 1
		}
	}
	if len(sortFields) > 0 {
		pipeline = append(pipeline, map[string]interface{}{"$sort": sortFields})
	}
	
	if a.Size > 0 {
		pipeline = append(pipeline, map[string]interface{}{"$limit": a.Size})
	}
	
	return map[string]interface{}{
		"$facet": map[string]interface{}{
			a.AggName: pipeline,
		},
	}, nil
}