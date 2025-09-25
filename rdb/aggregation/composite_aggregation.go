package aggregation

import (
	"fmt"
	"strings"
)

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
	} else {
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
