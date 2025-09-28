package aggregation

import (
	"reflect"
	"testing"
)

func TestCountAggregation_ToSQL(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{
			name:     "count all",
			field:    "",
			expected: "COUNT(*) AS doc_count",
		},
		{
			name:     "count field",
			field:    "status",
			expected: "COUNT(status) AS doc_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &CountAggregation{
				MetricAggregation: MetricAggregation{
					AggName: "doc_count",
					Field:   tt.field,
				},
			}

			sql, args, err := agg.ToSQL()
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("Expected SQL %s, got %s", tt.expected, sql)
			}
			if len(args) != 0 {
				t.Errorf("Expected no args, got %v", args)
			}
		})
	}
}

func TestCountAggregation_ToMongo(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected map[string]interface{}
	}{
		{
			name:  "count all",
			field: "",
			expected: map[string]interface{}{
				"$sum": 1,
			},
		},
		{
			name:  "count field",
			field: "status",
			expected: map[string]interface{}{
				"$sum": map[string]interface{}{
					"$cond": map[string]interface{}{
						"if": map[string]interface{}{
							"$and": []interface{}{
								// 字段不为null
								map[string]interface{}{"$ne": []interface{}{"$status", nil}},
								// 字段存在（不是missing）
								map[string]interface{}{"$ne": []interface{}{"$status", "$$REMOVE"}},
								// 如果是字符串类型，还要检查不为空字符串
								map[string]interface{}{
									"$cond": map[string]interface{}{
										"if":   map[string]interface{}{"$eq": []interface{}{map[string]interface{}{"$type": "$status"}, "string"}},
										"then": map[string]interface{}{"$ne": []interface{}{"$status", ""}},
										"else": true,
									},
								},
							},
						},
						"then": 1,
						"else": 0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &CountAggregation{
				MetricAggregation: MetricAggregation{
					AggName: "doc_count",
					Field:   tt.field,
				},
			}

			result, err := agg.ToMongo()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}