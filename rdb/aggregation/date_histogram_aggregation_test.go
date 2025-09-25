package aggregation

import (
	"strings"
	"testing"
)

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