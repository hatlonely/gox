package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的结构体
type TestESUser struct {
	ID       string    `rdb:"id,primary"`
	Name     string    `rdb:"name,required"`
	Email    string    `rdb:"email,unique"`
	Age      int       `rdb:"age"`
	Active   bool      `rdb:"active"`
	Score    float64   `rdb:"score"`
	Tags     []string  `rdb:"tags"`
	CreateAt time.Time `rdb:"create_at"`
}

func (u TestESUser) Table() string {
	return "users"
}

// 测试配置
var testESOptions = &ESOptions{
	Addresses:  []string{"http://localhost:9200"},
	Timeout:    30 * time.Second,
	MaxRetries: 3,
}

func TestNewESWithOptions(t *testing.T) {
	Convey("测试 NewESWithOptions 方法", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			Convey("使用完整配置创建连接", func() {
				es, err := NewESWithOptions(testESOptions)
				So(err, ShouldBeNil)
				So(es, ShouldNotBeNil)
				So(es.client, ShouldNotBeNil)
				So(es.builder, ShouldNotBeNil)

				// 清理资源
				es.Close()
			})

			Convey("使用认证配置", func() {
				options := &ESOptions{
					Addresses:  []string{"http://localhost:9200"},
					Username:   "elastic",
					Password:   "password",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
				}
				es, err := NewESWithOptions(options)
				So(err, ShouldBeNil)
				So(es, ShouldNotBeNil)

				// 清理资源
				es.Close()
			})

			Convey("使用 API Key 认证", func() {
				options := &ESOptions{
					Addresses:  []string{"http://localhost:9200"},
					APIKey:     "test-api-key",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
				}
				es, err := NewESWithOptions(options)
				So(err, ShouldBeNil)
				So(es, ShouldNotBeNil)

				// 清理资源
				es.Close()
			})

			Convey("连接不存在的服务器", func() {
				options := &ESOptions{
					Addresses:  []string{"http://non-existent-host:9200"},
					Timeout:    1 * time.Second, // 短超时
					MaxRetries: 1,
				}
				es, err := NewESWithOptions(options)
				So(err, ShouldNotBeNil)
				So(es, ShouldBeNil)
			})
		})
	})
}

func TestESRecord(t *testing.T) {
	Convey("测试 ESRecord 方法", t, func() {
		data := map[string]any{
			"_id":       "test_id_123",
			"id":        "user1",
			"name":      "John Doe",
			"email":     "john@example.com",
			"age":       30,
			"active":    true,
			"score":     95.5,
			"tags":      []string{"developer", "golang"},
			"create_at": time.Now(),
		}
		record := &ESRecord{
			id:     "test_id_123",
			index:  "users",
			source: data,
		}

		Convey("测试 Fields 方法", func() {
			fields := record.Fields()
			So(len(fields), ShouldEqual, len(data))
			So(fields["id"], ShouldEqual, "user1")
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
		})

		Convey("测试 Scan 方法", func() {
			var user TestESUser
			err := record.Scan(&user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, "user1")
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
			So(len(user.Tags), ShouldEqual, 2)
		})

		Convey("测试 ScanStruct 方法", func() {
			var user TestESUser
			err := record.ScanStruct(&user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, "user1")
			So(user.Name, ShouldEqual, "John Doe")
		})
	})
}

func TestESRecordBuilder(t *testing.T) {
	Convey("测试 ESRecordBuilder 方法", t, func() {
		builder := &ESRecordBuilder{}

		Convey("测试 FromStruct 方法", func() {
			user := TestESUser{
				ID:       "user1",
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      30,
				Active:   true,
				Score:    95.5,
				Tags:     []string{"developer", "golang"},
				CreateAt: time.Now(),
			}

			record := builder.FromStruct(user)
			So(record, ShouldNotBeNil)

			fields := record.Fields()
			So(fields["id"], ShouldEqual, "user1")
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
			So(fields["age"], ShouldEqual, 30)
			So(fields["active"], ShouldEqual, true)
			So(fields["score"], ShouldEqual, 95.5)
			So(len(fields["tags"].([]string)), ShouldEqual, 2)
		})

		Convey("测试 FromMap 方法", func() {
			data := map[string]any{
				"id":    "user1",
				"name":  "John Doe",
				"email": "john@example.com",
			}

			record := builder.FromMap(data, "users")
			So(record, ShouldNotBeNil)

			fields := record.Fields()
			So(fields["id"], ShouldEqual, "user1")
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
		})
	})
}

func TestESStructToMap(t *testing.T) {
	Convey("测试 esStructToMap 辅助函数", t, func() {
		Convey("正常结构体转换", func() {
			user := TestESUser{
				ID:    "user1",
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
				Tags:  []string{"developer"},
			}

			result := esStructToMap(user)
			So(result["id"], ShouldEqual, "user1")
			So(result["name"], ShouldEqual, "John Doe")
			So(result["email"], ShouldEqual, "john@example.com")
			So(result["age"], ShouldEqual, 30)
			So(len(result["tags"].([]string)), ShouldEqual, 1)
		})

		Convey("指针结构体转换", func() {
			user := &TestESUser{
				ID:   "user1",
				Name: "John Doe",
			}

			result := esStructToMap(user)
			So(result["id"], ShouldEqual, "user1")
			So(result["name"], ShouldEqual, "John Doe")
		})

		Convey("非结构体类型", func() {
			result := esStructToMap("not a struct")
			So(len(result), ShouldEqual, 0)
		})
	})
}

func TestESMapToStruct(t *testing.T) {
	Convey("测试 esMapToStruct 辅助函数", t, func() {
		Convey("正常转换", func() {
			data := map[string]any{
				"id":        "user1",
				"name":      "John Doe",
				"email":     "john@example.com",
				"age":       30,
				"active":    true,
				"score":     95.5,
				"tags":      []string{"developer", "golang"},
				"create_at": time.Now(),
			}

			var user TestESUser
			err := esMapToStruct(data, &user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, "user1")
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
			So(len(user.Tags), ShouldEqual, 2)
		})

		Convey("目标不是指针", func() {
			data := map[string]any{"id": "user1"}
			var user TestESUser
			err := esMapToStruct(data, user)
			So(err, ShouldNotBeNil)
		})

		Convey("目标不是结构体指针", func() {
			data := map[string]any{"value": 1}
			var value int
			err := esMapToStruct(data, &value)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestESMigrate(t *testing.T) {
	Convey("测试 ES Migrate 方法", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			Convey("创建带映射的索引", func() {
				model := &TableModel{
					Table: "test_users",
					Fields: []FieldDefinition{
						{Name: "id", Type: FieldTypeString, Required: true},
						{Name: "name", Type: FieldTypeString, Required: true},
						{Name: "email", Type: FieldTypeString},
						{Name: "age", Type: FieldTypeInt},
						{Name: "active", Type: FieldTypeBool, Default: true},
						{Name: "score", Type: FieldTypeFloat},
						{Name: "tags", Type: FieldTypeJSON},
						{Name: "create_at", Type: FieldTypeDate},
					},
					PrimaryKey: []string{"id"},
				}

				ctx := context.Background()
				err := es.Migrate(ctx, model)
				So(err, ShouldBeNil)

				// 清理测试索引
				es.DropTable(ctx, "test_users")
			})

			Convey("创建简单索引", func() {
				model := &TableModel{
					Table: "test_simple_index",
					Fields: []FieldDefinition{
						{Name: "id", Type: FieldTypeString, Required: true},
						{Name: "data", Type: FieldTypeJSON},
					},
					PrimaryKey: []string{"id"},
				}

				ctx := context.Background()
				err := es.Migrate(ctx, model)
				So(err, ShouldBeNil)

				// 清理测试索引
				es.DropTable(ctx, "test_simple_index")
			})
		})
	})
}

func TestESCRUDOperations(t *testing.T) {
	Convey("测试 ES CRUD 操作", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			// 创建测试索引
			ctx := context.Background()
			model := &TableModel{
				Table: "test_crud_users",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "email", Type: FieldTypeString},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "tags", Type: FieldTypeJSON},
					{Name: "create_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
			}
			es.Migrate(ctx, model)
			defer es.DropTable(ctx, "test_crud_users")

			Convey("测试 Create 方法", func() {
				user := TestESUser{
					ID:       "user1",
					Name:     "John Doe",
					Email:    "john@example.com",
					Age:      30,
					Active:   true,
					Score:    95.5,
					Tags:     []string{"developer", "golang"},
					CreateAt: time.Now(),
				}

				record := es.builder.FromStruct(user)
				err := es.Create(ctx, "test_crud_users", record)
				So(err, ShouldBeNil)
			})

			Convey("测试 Create 方法的 IgnoreConflict 选项", func() {
				// 先创建一条记录
				user := TestESUser{
					ID:       "user10",
					Name:     "Original User",
					Email:    "original@example.com",
					Age:      25,
					Active:   true,
					Score:    80.0,
					Tags:     []string{"original"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				err := es.Create(ctx, "test_crud_users", record)
				So(err, ShouldBeNil)

				// 尝试创建相同ID的记录，使用 IgnoreConflict 选项
				conflictUser := TestESUser{
					ID:       "user10",
					Name:     "Conflict User",
					Email:    "conflict@example.com",
					Age:      30,
					Active:   false,
					Score:    90.0,
					Tags:     []string{"conflict"},
					CreateAt: time.Now(),
				}
				conflictRecord := es.builder.FromStruct(conflictUser)

				// 使用 IgnoreConflict 选项，应该忽略冲突
				err = es.Create(ctx, "test_crud_users", conflictRecord, WithIgnoreConflict())
				So(err, ShouldBeNil)

				// 验证原始记录没有被修改
				pk := map[string]any{"_id": "user10"}
				result, err := es.Get(ctx, "test_crud_users", pk)
				So(err, ShouldBeNil)
				var retrievedUser TestESUser
				result.Scan(&retrievedUser)
				So(retrievedUser.Name, ShouldEqual, "Original User")
				So(retrievedUser.Email, ShouldEqual, "original@example.com")
			})

			Convey("测试 Create 方法的 UpdateOnConflict 选项", func() {
				// 先创建一条记录
				user := TestESUser{
					ID:       "user11",
					Name:     "Original User",
					Email:    "original11@example.com",
					Age:      25,
					Active:   true,
					Score:    80.0,
					Tags:     []string{"original"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				err := es.Create(ctx, "test_crud_users", record)
				So(err, ShouldBeNil)

				// 尝试创建相同ID的记录，使用 UpdateOnConflict 选项
				conflictUser := TestESUser{
					ID:       "user11",
					Name:     "Updated User",
					Email:    "updated11@example.com",
					Age:      30,
					Active:   false,
					Score:    90.0,
					Tags:     []string{"updated"},
					CreateAt: time.Now(),
				}
				conflictRecord := es.builder.FromStruct(conflictUser)

				// 使用 UpdateOnConflict 选项，应该更新记录
				err = es.Create(ctx, "test_crud_users", conflictRecord, WithUpdateOnConflict())
				So(err, ShouldBeNil)

				// 验证记录已被更新
				pk := map[string]any{"_id": "user11"}
				result, err := es.Get(ctx, "test_crud_users", pk)
				So(err, ShouldBeNil)
				var retrievedUser TestESUser
				result.Scan(&retrievedUser)
				So(retrievedUser.Name, ShouldEqual, "Updated User")
				So(retrievedUser.Email, ShouldEqual, "updated11@example.com")
				So(retrievedUser.Age, ShouldEqual, 30)
			})

			Convey("测试 Get 方法", func() {
				// 先创建一条记录
				user := TestESUser{
					ID:       "user2",
					Name:     "Jane Doe",
					Email:    "jane@example.com",
					Age:      25,
					Active:   true,
					Score:    88.5,
					Tags:     []string{"designer", "ui"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				es.Create(ctx, "test_crud_users", record)

				// 获取记录
				pk := map[string]any{"_id": "user2"}
				result, err := es.Get(ctx, "test_crud_users", pk)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)

				var retrievedUser TestESUser
				err = result.Scan(&retrievedUser)
				So(err, ShouldBeNil)
				So(retrievedUser.ID, ShouldEqual, "user2")
				So(retrievedUser.Name, ShouldEqual, "Jane Doe")
				So(retrievedUser.Email, ShouldEqual, "jane@example.com")
			})

			Convey("测试 Update 方法", func() {
				// 先创建一条记录
				user := TestESUser{
					ID:       "user3",
					Name:     "Bob Smith",
					Email:    "bob@example.com",
					Age:      35,
					Active:   true,
					Score:    92.0,
					Tags:     []string{"manager"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				es.Create(ctx, "test_crud_users", record)

				// 更新记录
				updatedUser := TestESUser{
					ID:       "user3",
					Name:     "Bob Smith Updated",
					Email:    "bob.updated@example.com",
					Age:      36,
					Active:   false,
					Score:    93.5,
					Tags:     []string{"manager", "senior"},
					CreateAt: time.Now(),
				}
				updatedRecord := es.builder.FromStruct(updatedUser)
				pk := map[string]any{"_id": "user3"}
				err := es.Update(ctx, "test_crud_users", pk, updatedRecord)
				So(err, ShouldBeNil)

				// 验证更新
				result, err := es.Get(ctx, "test_crud_users", pk)
				So(err, ShouldBeNil)
				var retrievedUser TestESUser
				result.Scan(&retrievedUser)
				So(retrievedUser.Name, ShouldEqual, "Bob Smith Updated")
				So(retrievedUser.Email, ShouldEqual, "bob.updated@example.com")
				So(retrievedUser.Age, ShouldEqual, 36)
			})

			Convey("测试 Delete 方法", func() {
				// 先创建一条记录
				user := TestESUser{
					ID:       "user4",
					Name:     "Alice Johnson",
					Email:    "alice@example.com",
					Age:      28,
					Active:   true,
					Score:    87.5,
					Tags:     []string{"analyst"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				es.Create(ctx, "test_crud_users", record)

				// 删除记录
				pk := map[string]any{"_id": "user4"}
				err := es.Delete(ctx, "test_crud_users", pk)
				So(err, ShouldBeNil)

				// 验证删除
				_, err = es.Get(ctx, "test_crud_users", pk)
				So(err, ShouldEqual, ErrRecordNotFound)
			})
		})
	})
}

func TestESFind(t *testing.T) {
	Convey("测试 ES Find 方法", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			// 创建测试索引和数据
			ctx := context.Background()

			// 使用动态表名避免冲突
			tableName := fmt.Sprintf("test_find_users_%d", time.Now().UnixNano())
			defer es.DropTable(ctx, tableName)

			// 插入测试数据
			users := []TestESUser{
				{ID: "1", Name: "John", Age: 30, Active: true, Score: 95.0, Tags: []string{"developer"}, CreateAt: time.Now()},
				{ID: "2", Name: "Jane", Age: 25, Active: true, Score: 88.0, Tags: []string{"designer"}, CreateAt: time.Now()},
				{ID: "3", Name: "Bob", Age: 35, Active: false, Score: 92.0, Tags: []string{"manager"}, CreateAt: time.Now()},
				{ID: "4", Name: "Alice", Age: 28, Active: true, Score: 90.0, Tags: []string{"analyst"}, CreateAt: time.Now()},
			}
			for _, user := range users {
				record := es.builder.FromStruct(user)
				err := es.Create(ctx, tableName, record)
				So(err, ShouldBeNil)
			}

			// 等待索引刷新
			time.Sleep(1 * time.Second)

			Convey("使用 TermQuery 查询", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				results, err := es.Find(ctx, tableName, termQuery)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 3) // John, Jane, Alice
			})

			Convey("使用 MatchQuery 查询", func() {
				matchQuery := &query.MatchQuery{Field: "name", Value: "Jo"}
				results, err := es.Find(ctx, tableName, matchQuery)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1) // John
			})

			Convey("带排序的查询", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				options := &QueryOptions{OrderBy: "age", OrderDesc: false}
				results, err := es.Find(ctx, tableName, termQuery, func(opts *QueryOptions) {
					*opts = *options
				})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 3)

				// 验证排序 (Jane:25, Alice:28, John:30)
				var firstUser TestESUser
				results[0].Scan(&firstUser)
				So(firstUser.Age, ShouldEqual, 25) // Jane
			})

			Convey("带分页的查询", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				options := &QueryOptions{Limit: 2, Offset: 1}
				results, err := es.Find(ctx, tableName, termQuery, func(opts *QueryOptions) {
					*opts = *options
				})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2)
			})
		})
	})
}

func TestESAggregate(t *testing.T) {
	Convey("测试 ES Aggregate 方法", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			// 创建测试索引和数据
			ctx := context.Background()
			model := &TableModel{
				Table: "test_agg_users",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "age", Type: FieldTypeInt},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "active", Type: FieldTypeBool, Default: true},
				},
				PrimaryKey: []string{"id"},
			}
			es.Migrate(ctx, model)
			defer es.DropTable(ctx, "test_agg_users")

			// 插入测试数据
			users := []TestESUser{
				{ID: "1", Name: "John", Age: 30, Score: 95.5, Active: true},
				{ID: "2", Name: "Jane", Age: 25, Score: 88.0, Active: true},
				{ID: "3", Name: "Bob", Age: 35, Score: 92.5, Active: false},
				{ID: "4", Name: "Alice", Age: 28, Score: 90.0, Active: true},
			}
			for _, user := range users {
				record := es.builder.FromStruct(user)
				es.Create(ctx, "test_agg_users", record)
			}

			// 等待索引刷新
			time.Sleep(1 * time.Second)

			Convey("Count 聚合", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				countAgg := &aggregation.CountAggregation{}
				countAgg.AggName = "total_count"
				countAgg.Field = "id"

				aggs := []aggregation.Aggregation{countAgg}
				result, err := es.Aggregate(ctx, "test_agg_users", termQuery, aggs)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("Avg 聚合", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				avgAgg := &aggregation.AvgAggregation{}
				avgAgg.AggName = "avg_score"
				avgAgg.Field = "score"

				aggs := []aggregation.Aggregation{avgAgg}
				result, err := es.Aggregate(ctx, "test_agg_users", termQuery, aggs)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("Terms 聚合", func() {
				termQuery := &query.TermQuery{Field: "active", Value: true}
				termsAgg := &aggregation.TermsAggregation{}
				termsAgg.AggName = "age_groups"
				termsAgg.Field = "age"

				aggs := []aggregation.Aggregation{termsAgg}
				result, err := es.Aggregate(ctx, "test_agg_users", termQuery, aggs)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})
		})
	})
}

func TestESBatchOperations(t *testing.T) {
	Convey("测试 ES 批量操作", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			// 创建测试索引
			ctx := context.Background()
			model := &TableModel{
				Table: "test_batch_users",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "tags", Type: FieldTypeJSON},
					{Name: "create_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
			}
			es.Migrate(ctx, model)
			defer es.DropTable(ctx, "test_batch_users")

			Convey("测试 BatchCreate", func() {
				users := []TestESUser{
					{ID: "batch1", Name: "User1", Age: 20, Tags: []string{"tag1"}, CreateAt: time.Now()},
					{ID: "batch2", Name: "User2", Age: 21, Tags: []string{"tag2"}, CreateAt: time.Now()},
					{ID: "batch3", Name: "User3", Age: 22, Tags: []string{"tag3"}, CreateAt: time.Now()},
				}

				var records []Record
				for _, user := range users {
					records = append(records, es.builder.FromStruct(user))
				}

				err := es.BatchCreate(ctx, "test_batch_users", records)
				So(err, ShouldBeNil)
			})

			Convey("测试 BatchCreate 的 CreateOption", func() {
				// 先创建一些记录
				users := []TestESUser{
					{ID: "batch50", Name: "OriginalUser50", Age: 20, CreateAt: time.Now()},
					{ID: "batch51", Name: "OriginalUser51", Age: 21, CreateAt: time.Now()},
				}
				var records []Record
				for _, user := range users {
					records = append(records, es.builder.FromStruct(user))
				}
				err := es.BatchCreate(ctx, "test_batch_users", records)
				So(err, ShouldBeNil)

				// 测试批量创建时使用 IgnoreConflict 选项
				conflictUsers := []TestESUser{
					{ID: "batch50", Name: "ConflictUser50", Age: 30, CreateAt: time.Now()}, // 冲突记录
					{ID: "batch52", Name: "NewUser52", Age: 22, CreateAt: time.Now()},      // 新记录
				}
				var conflictRecords []Record
				for _, user := range conflictUsers {
					conflictRecords = append(conflictRecords, es.builder.FromStruct(user))
				}

				err = es.BatchCreate(ctx, "test_batch_users", conflictRecords, WithIgnoreConflict())
				So(err, ShouldBeNil)

				// 验证ID=batch50的记录没有被修改，ID=batch52的记录被创建
				pk50 := map[string]any{"_id": "batch50"}
				result50, err := es.Get(ctx, "test_batch_users", pk50)
				So(err, ShouldBeNil)
				var user50 TestESUser
				result50.Scan(&user50)
				So(user50.Name, ShouldEqual, "OriginalUser50") // 原始记录没有被修改

				pk52 := map[string]any{"_id": "batch52"}
				result52, err := es.Get(ctx, "test_batch_users", pk52)
				So(err, ShouldBeNil)
				var user52 TestESUser
				result52.Scan(&user52)
				So(user52.Name, ShouldEqual, "NewUser52") // 新记录被创建
			})

			Convey("测试 BatchUpdate", func() {
				// 先创建记录
				users := []TestESUser{
					{ID: "batch4", Name: "User4", Age: 23, CreateAt: time.Now()},
					{ID: "batch5", Name: "User5", Age: 24, CreateAt: time.Now()},
				}
				var records []Record
				for _, user := range users {
					records = append(records, es.builder.FromStruct(user))
				}
				es.BatchCreate(ctx, "test_batch_users", records)

				// 批量更新
				updatedUsers := []TestESUser{
					{ID: "batch4", Name: "Updated User4", Age: 33, CreateAt: time.Now()},
					{ID: "batch5", Name: "Updated User5", Age: 34, CreateAt: time.Now()},
				}
				var updatedRecords []Record
				var pks []map[string]any
				for _, user := range updatedUsers {
					updatedRecords = append(updatedRecords, es.builder.FromStruct(user))
					pks = append(pks, map[string]any{"_id": user.ID})
				}

				err := es.BatchUpdate(ctx, "test_batch_users", pks, updatedRecords)
				So(err, ShouldBeNil)
			})

			Convey("测试 BatchDelete", func() {
				// 先创建记录
				users := []TestESUser{
					{ID: "batch6", Name: "User6", Age: 25, CreateAt: time.Now()},
					{ID: "batch7", Name: "User7", Age: 26, CreateAt: time.Now()},
				}
				var records []Record
				for _, user := range users {
					records = append(records, es.builder.FromStruct(user))
				}
				es.BatchCreate(ctx, "test_batch_users", records)

				// 批量删除
				pks := []map[string]any{
					{"_id": "batch6"},
					{"_id": "batch7"},
				}

				err := es.BatchDelete(ctx, "test_batch_users", pks)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestESTransaction(t *testing.T) {
	Convey("测试 ES 事务操作", t, func() {
		SkipConvey("跳过 ES 测试 - 需要运行中的 Elasticsearch 实例", func() {
			es, err := NewESWithOptions(testESOptions)
			So(err, ShouldBeNil)
			defer es.Close()

			// 创建测试索引
			ctx := context.Background()
			model := &TableModel{
				Table: "test_tx_users",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "tags", Type: FieldTypeJSON},
					{Name: "create_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
			}
			es.Migrate(ctx, model)
			defer es.DropTable(ctx, "test_tx_users")

			Convey("测试 BeginTx 和手动提交", func() {
				tx, err := es.BeginTx(ctx)
				So(err, ShouldBeNil)
				So(tx, ShouldNotBeNil)

				user := TestESUser{ID: "tx1", Name: "TxUser1", Age: 30, CreateAt: time.Now()}
				record := es.builder.FromStruct(user)
				err = tx.Create(ctx, "test_tx_users", record)
				So(err, ShouldBeNil)

				err = tx.Commit()
				So(err, ShouldBeNil)

				// 验证提交成功
				pk := map[string]any{"_id": "tx1"}
				result, err := es.Get(ctx, "test_tx_users", pk)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("测试事务回滚", func() {
				tx, err := es.BeginTx(ctx)
				So(err, ShouldBeNil)

				user := TestESUser{ID: "tx2", Name: "TxUser2", Age: 25, CreateAt: time.Now()}
				record := es.builder.FromStruct(user)
				err = tx.Create(ctx, "test_tx_users", record)
				So(err, ShouldBeNil)

				err = tx.Rollback()
				So(err, ShouldBeNil)

				// 验证回滚成功（ES的伪事务回滚只是清空操作队列）
				pk := map[string]any{"_id": "tx2"}
				_, err = es.Get(ctx, "test_tx_users", pk)
				So(err, ShouldEqual, ErrRecordNotFound)
			})

			Convey("测试 WithTx", func() {
				err := es.WithTx(ctx, func(tx Transaction) error {
					user := TestESUser{ID: "tx3", Name: "TxUser3", Age: 28, CreateAt: time.Now()}
					record := es.builder.FromStruct(user)
					return tx.Create(ctx, "test_tx_users", record)
				})
				So(err, ShouldBeNil)

				// 验证提交成功
				pk := map[string]any{"_id": "tx3"}
				result, err := es.Get(ctx, "test_tx_users", pk)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("测试事务中的 CreateOption", func() {
				tx, err := es.BeginTx(ctx)
				So(err, ShouldBeNil)
				defer tx.Rollback()

				// 先创建一条记录
				user := TestESUser{
					ID:       "tx100",
					Name:     "TxOriginal",
					Email:    "txoriginal@example.com",
					Age:      25,
					Active:   true,
					Score:    85.0,
					Tags:     []string{"original"},
					CreateAt: time.Now(),
				}
				record := es.builder.FromStruct(user)
				err = tx.Create(ctx, "test_tx_users", record)
				So(err, ShouldBeNil)

				// 测试 IgnoreConflict 选项
				conflictUser := TestESUser{
					ID:       "tx100",
					Name:     "TxConflict",
					Email:    "txconflict@example.com",
					Age:      30,
					Active:   false,
					Score:    95.0,
					Tags:     []string{"conflict"},
					CreateAt: time.Now(),
				}
				conflictRecord := es.builder.FromStruct(conflictUser)

				err = tx.Create(ctx, "test_tx_users", conflictRecord, WithIgnoreConflict())
				So(err, ShouldBeNil)

				// 测试 UpdateOnConflict 选项
				updateUser := TestESUser{
					ID:       "tx100",
					Name:     "TxUpdated",
					Email:    "txupdated@example.com",
					Age:      35,
					Active:   false,
					Score:    99.0,
					Tags:     []string{"updated"},
					CreateAt: time.Now(),
				}
				updateRecord := es.builder.FromStruct(updateUser)

				err = tx.Create(ctx, "test_tx_users", updateRecord, WithUpdateOnConflict())
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestESGetBuilder(t *testing.T) {
	Convey("测试 ES GetBuilder 方法", t, func() {
		es := &ES{
			builder: &ESRecordBuilder{},
		}

		builder := es.GetBuilder()
		So(builder, ShouldNotBeNil)
		So(builder, ShouldEqual, es.builder)
	})
}

func TestESClose(t *testing.T) {
	Convey("测试 ES Close 方法", t, func() {
		es := &ES{
			builder: &ESRecordBuilder{},
		}

		err := es.Close()
		So(err, ShouldBeNil)
	})
}

func TestESFieldTypeMapping(t *testing.T) {
	Convey("测试 ES 字段类型映射", t, func() {
		es := &ES{}

		Convey("测试 mapFieldTypeToES", func() {
			// String 类型
			stringMapping := es.mapFieldTypeToES(FieldTypeString, 100)
			So(stringMapping["type"], ShouldEqual, "text")
			So(stringMapping["fields"], ShouldNotBeNil)

			// Int 类型
			intMapping := es.mapFieldTypeToES(FieldTypeInt, 0)
			So(intMapping["type"], ShouldEqual, "long")

			// Float 类型
			floatMapping := es.mapFieldTypeToES(FieldTypeFloat, 0)
			So(floatMapping["type"], ShouldEqual, "double")

			// Bool 类型
			boolMapping := es.mapFieldTypeToES(FieldTypeBool, 0)
			So(boolMapping["type"], ShouldEqual, "boolean")

			// Date 类型
			dateMapping := es.mapFieldTypeToES(FieldTypeDate, 0)
			So(dateMapping["type"], ShouldEqual, "date")
			So(dateMapping["format"], ShouldNotBeNil)

			// JSON 类型
			jsonMapping := es.mapFieldTypeToES(FieldTypeJSON, 0)
			So(jsonMapping["type"], ShouldEqual, "object")
		})

		Convey("测试 buildIndexMapping", func() {
			model := &TableModel{
				Table: "test_mapping",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "data", Type: FieldTypeJSON},
					{Name: "created_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
			}

			mapping := es.buildIndexMapping(model)
			So(mapping["mappings"], ShouldNotBeNil)
			So(mapping["settings"], ShouldNotBeNil)

			mappings := mapping["mappings"].(map[string]any)
			properties := mappings["properties"].(map[string]any)

			So(len(properties), ShouldEqual, 7)
			So(properties["id"], ShouldNotBeNil)
			So(properties["name"], ShouldNotBeNil)
			So(properties["age"], ShouldNotBeNil)
			So(properties["active"], ShouldNotBeNil)
			So(properties["score"], ShouldNotBeNil)
			So(properties["data"], ShouldNotBeNil)
			So(properties["created_at"], ShouldNotBeNil)
		})
	})
}