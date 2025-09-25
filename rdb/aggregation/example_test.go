package aggregation_test

import (
	"fmt"
	"log"

	"github.com/hatlonely/gox/rdb/aggregation"
)

// 演示如何使用各种聚合类型
func ExampleAggregations() {
	// 1. 指标聚合示例
	sumAgg := &aggregation.SumAggregation{
		MetricAggregation: aggregation.MetricAggregation{
			AggName: "total_sales",
			Field:   "amount",
		},
	}
	
	avgAgg := &aggregation.AvgAggregation{
		MetricAggregation: aggregation.MetricAggregation{
			AggName: "avg_price",
			Field:   "price",
		},
	}
	
	// 演示 Elasticsearch 查询生成
	fmt.Printf("Sum aggregation ES query: %v\n", sumAgg.ToES())
	
	// 演示 SQL 查询生成
	if sql, args, err := avgAgg.ToSQL(); err == nil {
		fmt.Printf("Avg aggregation SQL: %s (args: %v)\n", sql, args)
	}
	
	// 2. 桶聚合示例
	termsAgg := &aggregation.TermsAggregation{
		BucketAggregation: aggregation.BucketAggregation{
			AggName: "by_status",
			Field:   "status",
			SubAggregations: []aggregation.Aggregation{sumAgg, avgAgg},
		},
		Size:  10,
		Order: map[string]string{"total_sales": "desc"},
	}
	
	fmt.Printf("Terms aggregation with sub-aggregations: %v\n", termsAgg.Type())
	
	// 3. 日期直方图聚合示例
	dateHistoAgg := &aggregation.DateHistogramAggregation{
		BucketAggregation: aggregation.BucketAggregation{
			AggName: "monthly_trend",
			Field:   "created_at",
			SubAggregations: []aggregation.Aggregation{
				&aggregation.CountAggregation{
					MetricAggregation: aggregation.MetricAggregation{
						AggName: "doc_count",
						Field:   "",
					},
				},
			},
		},
		Interval: "1M",
		Format:   "yyyy-MM",
	}
	
	if sql, args, err := dateHistoAgg.ToSQL(); err == nil {
		fmt.Printf("Date histogram SQL: %s (args: %v)\n", sql, args)
	}
	
	// 4. 复合聚合示例
	compositeAgg := &aggregation.CompositeAggregation{
		BucketAggregation: aggregation.BucketAggregation{
			AggName: "region_category",
		},
		Sources: []aggregation.CompositeSource{
			{Name: "region", Field: "customer.region", Order: "asc"},
			{Name: "category", Field: "product.category", Order: "desc"},
		},
		Size: 100,
	}
	
	fmt.Printf("Composite aggregation sources count: %d\n", len(compositeAgg.Sources))
	
	// 5. 聚合结果处理示例
	result := aggregation.NewAggregationResult()
	result.SetResult("total_sales", 15000.50)
	result.SetResult("order_count", int64(150))
	
	// 创建桶数据
	bucket1 := aggregation.NewBucket("pending", 10)
	bucket1.SetSubAggregation("avg_amount", 120.5)
	
	bucket2 := aggregation.NewBucket("completed", 140)
	bucket2.SetSubAggregation("avg_amount", 180.0)
	
	buckets := []aggregation.Bucket{bucket1, bucket2}
	result.SetResult("by_status", buckets)
	
	// 读取聚合结果
	fmt.Printf("Total sales: %.2f\n", result.GetValue("total_sales"))
	fmt.Printf("Order count: %d\n", result.GetCount("order_count"))
	
	// 处理桶聚合结果
	statusBuckets := result.GetBuckets("by_status")
	for _, bucket := range statusBuckets {
		avgAmount := bucket.SubAggregations().GetValue("avg_amount")
		fmt.Printf("Status: %v, Count: %d, Avg Amount: %.2f\n",
			bucket.Key(), bucket.DocCount(), avgAmount)
	}
}

// 演示复杂的嵌套聚合场景
func ExampleComplexAggregation() {
	// 场景：按地区分组，每个地区内按产品类别分组，计算各种指标
	regionAgg := &aggregation.TermsAggregation{
		BucketAggregation: aggregation.BucketAggregation{
			AggName: "by_region",
			Field:   "customer.region",
			SubAggregations: []aggregation.Aggregation{
				// 地区内按类别聚合
				&aggregation.TermsAggregation{
					BucketAggregation: aggregation.BucketAggregation{
						AggName: "by_category",
						Field:   "product.category",
						SubAggregations: []aggregation.Aggregation{
							// 每个类别的销售指标
							&aggregation.SumAggregation{
								MetricAggregation: aggregation.MetricAggregation{
									AggName: "total_revenue",
									Field:   "amount",
								},
							},
							&aggregation.AvgAggregation{
								MetricAggregation: aggregation.MetricAggregation{
									AggName: "avg_order_value",
									Field:   "amount",
								},
							},
							&aggregation.CountAggregation{
								MetricAggregation: aggregation.MetricAggregation{
									AggName: "order_count",
									Field:   "",
								},
							},
						},
					},
					Size: 5, // 每个地区取前5个类别
				},
				// 地区总销售额
				&aggregation.SumAggregation{
					MetricAggregation: aggregation.MetricAggregation{
						AggName: "region_total",
						Field:   "amount",
					},
				},
			},
		},
		Size:  10, // 取前10个地区
		Order: map[string]string{"region_total": "desc"},
	}
	
	// 演示生成的查询结构
	esQuery := regionAgg.ToES()
	fmt.Printf("Complex aggregation type: %v\n", regionAgg.Type())
	
	// 检查是否包含子聚合
	if terms, ok := esQuery["terms"]; ok {
		fmt.Printf("Field: %v\n", terms.(map[string]interface{})["field"])
	}
	
	if aggs, ok := esQuery["aggs"]; ok {
		aggregations := aggs.(map[string]interface{})
		fmt.Printf("Sub-aggregations count: %d\n", len(aggregations))
		
		// 检查嵌套结构
		if byCategoryAgg, exists := aggregations["by_category"]; exists {
			if categoryTerms, ok := byCategoryAgg.(map[string]interface{})["terms"]; ok {
				fmt.Printf("Category field: %v\n", categoryTerms.(map[string]interface{})["field"])
			}
		}
	}
}

func init() {
	// 运行示例
	fmt.Println("=== Aggregation Examples ===")
	ExampleAggregations()
	
	fmt.Println("\n=== Complex Aggregation Example ===")
	ExampleComplexAggregation()
}

// 演示如何在实际业务中使用聚合（伪代码）
func ExampleBusinessUsage() {
	// 这个例子展示如何在业务代码中使用聚合接口
	
	// 假设有一个 RDB 实例
	// rdb := getSomeRDBInstance()
	// ctx := context.Background()
	
	// 构建查询条件
	// query := &query.RangeQuery{
	//     Field: "created_at",
	//     From:  "2024-01-01",
	//     To:    "2024-12-31",
	// }
	
	// 构建聚合
	aggs := []aggregation.Aggregation{
		&aggregation.TermsAggregation{
			BucketAggregation: aggregation.BucketAggregation{
				AggName: "by_status",
				Field:   "status",
				SubAggregations: []aggregation.Aggregation{
					&aggregation.SumAggregation{
						MetricAggregation: aggregation.MetricAggregation{
							AggName: "total_amount",
							Field:   "amount",
						},
					},
				},
			},
			Size: 10,
		},
	}
	
	// 执行聚合查询
	// result, err := rdb.Aggregate(ctx, "orders", query, aggs)
	// if err != nil {
	//     log.Fatal(err)
	// }
	
	// 处理结果
	// buckets := result.GetBuckets("by_status")
	// for _, bucket := range buckets {
	//     totalAmount := bucket.SubAggregations().GetValue("total_amount")
	//     fmt.Printf("Status: %v, Orders: %d, Total: %.2f\n",
	//         bucket.Key(), bucket.DocCount(), totalAmount)
	// }
	
	log.Printf("Aggregation count: %d", len(aggs))
}