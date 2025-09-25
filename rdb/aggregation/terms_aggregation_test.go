package aggregation

import (
	"strings"
	"testing"
)

func TestTermsAggregation_ToES(t *testing.T) {
	subAgg := &SumAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "total_amount",
			Field:   "amount",
		},
	}

	agg := &TermsAggregation{
		BucketAggregation: BucketAggregation{
			AggName:         "by_status",
			Field:           "status",
			SubAggregations: []Aggregation{subAgg},
		},
		Size:  10,
		Order: map[string]string{"_count": "desc"},
	}

	result := agg.ToES()
	
	if result["terms"] == nil {
		t.Error("Expected 'terms' key in result")
	}
	
	terms := result["terms"].(map[string]interface{})
	if terms["field"] != "status" {
		t.Errorf("Expected field 'status', got %v", terms["field"])
	}
	if terms["size"] != 10 {
		t.Errorf("Expected size 10, got %v", terms["size"])
	}
	
	if result["aggs"] == nil {
		t.Error("Expected 'aggs' key in result")
	}
}

func TestTermsAggregation_ToSQL(t *testing.T) {
	subAgg := &SumAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "total_amount",
			Field:   "amount",
		},
	}

	agg := &TermsAggregation{
		BucketAggregation: BucketAggregation{
			AggName:         "by_status",
			Field:           "status",
			SubAggregations: []Aggregation{subAgg},
		},
		Size:  10,
		Order: map[string]string{"total_amount": "desc"},
	}

	sql, args, err := agg.ToSQL()
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	expectedParts := []string{
		"GROUP BY status",
		"SUM(amount) AS total_amount",
		"ORDER BY total_amount DESC",
		"LIMIT 10",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(sql, part) {
			t.Errorf("Expected SQL to contain '%s', got: %s", part, sql)
		}
	}
	
	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}