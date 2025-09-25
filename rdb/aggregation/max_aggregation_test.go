package aggregation

import (
	"testing"
)

func TestMaxAggregation_ToSQL(t *testing.T) {
	agg := &MaxAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "max_price",
			Field:   "price",
		},
	}

	expectedSQL := "MAX(price) AS max_price"
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