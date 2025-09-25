package aggregation

import (
	"strings"
	"testing"
)

func TestCompositeAggregation_ToES(t *testing.T) {
	agg := &CompositeAggregation{
		BucketAggregation: BucketAggregation{
			AggName: "multi_terms",
		},
		Sources: []CompositeSource{
			{Name: "region", Field: "region", Order: "asc"},
			{Name: "category", Field: "category", Order: "desc"},
		},
		Size:  100,
		After: map[string]interface{}{"region": "beijing", "category": "electronics"},
	}

	result := agg.ToES()
	
	if result["composite"] == nil {
		t.Error("Expected 'composite' key in result")
	}
	
	composite := result["composite"].(map[string]interface{})
	
	if composite["sources"] == nil {
		t.Error("Expected 'sources' key in composite")
	}
	
	sources := composite["sources"].([]map[string]interface{})
	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}
	
	if composite["size"] != 100 {
		t.Errorf("Expected size 100, got %v", composite["size"])
	}
	
	if composite["after"] == nil {
		t.Error("Expected 'after' key in composite")
	}
}

func TestCompositeAggregation_ToSQL(t *testing.T) {
	subAgg := &AvgAggregation{
		MetricAggregation: MetricAggregation{
			AggName: "avg_price",
			Field:   "price",
		},
	}

	agg := &CompositeAggregation{
		BucketAggregation: BucketAggregation{
			AggName:         "multi_group",
			SubAggregations: []Aggregation{subAgg},
		},
		Sources: []CompositeSource{
			{Name: "region", Field: "region", Order: "asc"},
			{Name: "category", Field: "category", Order: "desc"},
		},
		Size: 50,
	}

	sql, args, err := agg.ToSQL()
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	expectedParts := []string{
		"GROUP BY region, category",
		"AVG(price) AS avg_price",
		"ORDER BY region ASC, category DESC",
		"LIMIT 50",
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