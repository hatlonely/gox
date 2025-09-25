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

func TestDateHistogramAggregation_ToES(t *testing.T) {
	agg := &DateHistogramAggregation{
		BucketAggregation: BucketAggregation{
			AggName: "by_month",
			Field:   "created_at",
		},
		Interval: "1M",
		Format:   "yyyy-MM",
		TimeZone: "UTC",
	}

	result := agg.ToES()
	
	if result["date_histogram"] == nil {
		t.Error("Expected 'date_histogram' key in result")
	}
	
	dateHisto := result["date_histogram"].(map[string]interface{})
	if dateHisto["field"] != "created_at" {
		t.Errorf("Expected field 'created_at', got %v", dateHisto["field"])
	}
	if dateHisto["interval"] != "1M" {
		t.Errorf("Expected interval '1M', got %v", dateHisto["interval"])
	}
	if dateHisto["format"] != "yyyy-MM" {
		t.Errorf("Expected format 'yyyy-MM', got %v", dateHisto["format"])
	}
	if dateHisto["time_zone"] != "UTC" {
		t.Errorf("Expected timezone 'UTC', got %v", dateHisto["time_zone"])
	}
}

func TestDateHistogramAggregation_ToSQL(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		expected string
	}{
		{
			name:     "daily",
			interval: "1d",
			expected: "DATE(created_at)",
		},
		{
			name:     "monthly",
			interval: "1M",
			expected: "DATE_FORMAT(created_at, '%Y-%m-01')",
		},
		{
			name:     "yearly",
			interval: "1y",
			expected: "DATE_FORMAT(created_at, '%Y-01-01')",
		},
		{
			name:     "hourly",
			interval: "1h",
			expected: "DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &DateHistogramAggregation{
				BucketAggregation: BucketAggregation{
					AggName: "by_time",
					Field:   "created_at",
				},
				Interval: tt.interval,
			}

			sql, args, err := agg.ToSQL()
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !strings.Contains(sql, tt.expected) {
				t.Errorf("Expected SQL to contain '%s', got: %s", tt.expected, sql)
			}
			
			if len(args) != 0 {
				t.Errorf("Expected no args, got %v", args)
			}
		})
	}
}

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