package aggregation

import (
	"reflect"
	"testing"
)

func TestSumAggregation_ToES(t *testing.T) {
	agg := &SumAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "total_sales",
			Field:   "amount",
		},
	}

	expected := map[string]interface{}{
		"sum": map[string]interface{}{
			"field": "amount",
		},
	}

	result := agg.ToES()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestSumAggregation_ToSQL(t *testing.T) {
	agg := &SumAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "total_sales",
			Field:   "amount",
		},
	}

	expectedSQL := "SUM(amount) AS total_sales"
	sql, args, err := agg.ToSQL()
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if sql != expectedSQL {
		t.Errorf("Expected SQL %s, got %s", expectedSQL, sql)
	}
	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSumAggregation_ToMongo(t *testing.T) {
	agg := &SumAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "total_sales",
			Field:   "amount",
		},
	}

	expected := map[string]interface{}{
		"$sum": "$amount",
	}

	result, err := agg.ToMongo()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}