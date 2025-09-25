package aggregation

import (
	"fmt"
	"strings"
)

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
	} else {
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
