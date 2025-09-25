package aggregation

import (
	"reflect"
	"testing"
)

func TestAvgAggregation_ToES(t *testing.T) {
	agg := &AvgAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "avg_amount",
			Field:   "amount",
		},
	}

	expected := map[string]interface{}{
		"avg": map[string]interface{}{
			"field": "amount",
		},
	}

	result := agg.ToES()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}