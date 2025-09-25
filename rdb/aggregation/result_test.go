package aggregation

import (
	"testing"
)

func TestDefaultAggregationResult(t *testing.T) {
	result := NewAggregationResult()
	
	// 测试设置和获取值
	result.SetResult("total_sales", 1500.50)
	result.SetResult("order_count", int64(25))
	
	// 测试 Get 方法
	if totalSales := result.Get("total_sales"); totalSales != 1500.50 {
		t.Errorf("Expected 1500.50, got %v", totalSales)
	}
	
	// 测试 GetValue 方法（支持类型转换）
	if value := result.GetValue("total_sales"); value != 1500.50 {
		t.Errorf("Expected 1500.50, got %v", value)
	}
	
	if value := result.GetValue("order_count"); value != 25.0 {
		t.Errorf("Expected 25.0, got %v", value)
	}
	
	// 测试 GetCount 方法
	if count := result.GetCount("order_count"); count != 25 {
		t.Errorf("Expected 25, got %v", count)
	}
	
	// 测试不存在的键
	if value := result.GetValue("nonexistent"); value != 0.0 {
		t.Errorf("Expected 0.0 for nonexistent key, got %v", value)
	}
}

func TestDefaultAggregationResult_Buckets(t *testing.T) {
	result := NewAggregationResult()
	
	// 创建桶数据
	bucket1 := NewBucket("pending", 10)
	bucket1.SetSubAggregation("avg_amount", 150.0)
	
	bucket2 := NewBucket("completed", 20)
	bucket2.SetSubAggregation("avg_amount", 200.0)
	
	buckets := []Bucket{bucket1, bucket2}
	result.SetResult("by_status", buckets)
	
	// 测试获取桶
	retrievedBuckets := result.GetBuckets("by_status")
	if len(retrievedBuckets) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(retrievedBuckets))
	}
	
	// 测试第一个桶
	if retrievedBuckets[0].Key() != "pending" {
		t.Errorf("Expected key 'pending', got %v", retrievedBuckets[0].Key())
	}
	if retrievedBuckets[0].DocCount() != 10 {
		t.Errorf("Expected doc count 10, got %v", retrievedBuckets[0].DocCount())
	}
	
	// 测试子聚合
	subResult := retrievedBuckets[0].SubAggregations()
	if avgAmount := subResult.GetValue("avg_amount"); avgAmount != 150.0 {
		t.Errorf("Expected avg_amount 150.0, got %v", avgAmount)
	}
	
	// 测试不存在的桶聚合
	if nilBuckets := result.GetBuckets("nonexistent"); nilBuckets != nil {
		t.Errorf("Expected nil for nonexistent bucket aggregation, got %v", nilBuckets)
	}
}

func TestDefaultBucket(t *testing.T) {
	bucket := NewBucket("electronics", 100)
	
	// 测试基本属性
	if bucket.Key() != "electronics" {
		t.Errorf("Expected key 'electronics', got %v", bucket.Key())
	}
	
	if bucket.DocCount() != 100 {
		t.Errorf("Expected doc count 100, got %v", bucket.DocCount())
	}
	
	// 测试子聚合
	bucket.SetSubAggregation("total_revenue", 5000.0)
	bucket.SetSubAggregation("avg_price", 50.0)
	
	subAggs := bucket.SubAggregations()
	if revenue := subAggs.GetValue("total_revenue"); revenue != 5000.0 {
		t.Errorf("Expected total_revenue 5000.0, got %v", revenue)
	}
	
	if avgPrice := subAggs.GetValue("avg_price"); avgPrice != 50.0 {
		t.Errorf("Expected avg_price 50.0, got %v", avgPrice)
	}
}

func TestAggregationResult_TypeConversions(t *testing.T) {
	result := NewAggregationResult()
	
	// 测试不同数值类型的转换
	result.SetResult("int_value", 42)
	result.SetResult("int64_value", int64(100))
	result.SetResult("float64_value", 3.14)
	
	// GetValue 应该能转换所有数值类型为 float64
	if value := result.GetValue("int_value"); value != 42.0 {
		t.Errorf("Expected 42.0, got %v", value)
	}
	
	if value := result.GetValue("int64_value"); value != 100.0 {
		t.Errorf("Expected 100.0, got %v", value)
	}
	
	if value := result.GetValue("float64_value"); value != 3.14 {
		t.Errorf("Expected 3.14, got %v", value)
	}
	
	// GetCount 应该能转换整数类型为 int64
	if count := result.GetCount("int_value"); count != 42 {
		t.Errorf("Expected 42, got %v", count)
	}
	
	if count := result.GetCount("int64_value"); count != 100 {
		t.Errorf("Expected 100, got %v", count)
	}
	
	// 测试无效类型
	result.SetResult("string_value", "not a number")
	if value := result.GetValue("string_value"); value != 0.0 {
		t.Errorf("Expected 0.0 for string value, got %v", value)
	}
	
	if count := result.GetCount("string_value"); count != 0 {
		t.Errorf("Expected 0 for string value, got %v", count)
	}
}