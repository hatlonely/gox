package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的结构体
type TestMongoUser struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" rdb:"_id"`
	UserID   int                `bson:"user_id" rdb:"user_id"`
	Name     string             `bson:"name" rdb:"name"`
	Email    string             `bson:"email" rdb:"email"`
	Age      int                `bson:"age" rdb:"age"`
	Active   bool               `bson:"active" rdb:"active"`
	Score    float64            `bson:"score" rdb:"score"`
	CreateAt time.Time          `bson:"create_at" rdb:"create_at"`
}

// 测试配置
var testMongoOptions = &MongoOptions{
	Host:        "localhost",
	Port:        27017,
	Database:    "testdb",
	Username:    "admin",
	Password:    "admin123",
	AuthSource:  "admin",
	Timeout:     30 * time.Second,
	MaxPoolSize: 100,
	MinPoolSize: 0,
}

func TestNewMongoWithOptions(t *testing.T) {
	Convey("测试 NewMongoWithOptions 方法", t, func() {
		Convey("使用完整配置创建连接", func() {
			mongo, err := NewMongoWithOptions(testMongoOptions)
			So(err, ShouldBeNil)
			So(mongo, ShouldNotBeNil)
			So(mongo.client, ShouldNotBeNil)
			So(mongo.database, ShouldNotBeNil)
			So(mongo.builder, ShouldNotBeNil)
			So(mongo.dbName, ShouldEqual, "testdb")

			// 清理资源
			mongo.Close()
		})

		Convey("使用自定义 URI", func() {
			options := &MongoOptions{
				URI:     "mongodb://admin:admin123@localhost:27017/testdb?authSource=admin",
				Timeout: 30 * time.Second,
			}
			mongo, err := NewMongoWithOptions(options)
			So(err, ShouldBeNil)
			So(mongo, ShouldNotBeNil)

			// 清理资源
			mongo.Close()
		})

		Convey("使用无认证连接", func() {
			options := &MongoOptions{
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
				Timeout:  30 * time.Second,
			}
			// 注意：这可能会失败，因为测试环境需要认证
			mongo, err := NewMongoWithOptions(options)
			if err == nil {
				mongo.Close()
			}
			// 我们不断言错误，因为这取决于MongoDB配置
		})

		Convey("连接不存在的服务器", func() {
			options := &MongoOptions{
				Host:     "non-existent-host",
				Port:     27017,
				Database: "testdb",
				Timeout:  1 * time.Second, // 短超时
			}
			mongo, err := NewMongoWithOptions(options)
			So(err, ShouldNotBeNil)
			So(mongo, ShouldBeNil)
		})
	})
}

func TestMongoRecord(t *testing.T) {
	Convey("测试 MongoRecord 方法", t, func() {
		data := map[string]any{
			"_id":       primitive.NewObjectID(),
			"user_id":   1,
			"name":      "John Doe",
			"email":     "john@example.com",
			"age":       30,
			"active":    true,
			"score":     95.5,
			"create_at": time.Now(),
		}
		record := &MongoRecord{data: data}

		Convey("测试 Fields 方法", func() {
			fields := record.Fields()
			So(len(fields), ShouldEqual, len(data))
			So(fields["user_id"], ShouldEqual, 1)
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
		})

		Convey("测试 Scan 方法", func() {
			var user TestMongoUser
			err := record.Scan(&user)
			So(err, ShouldBeNil)
			So(user.UserID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
		})

		Convey("测试 ScanStruct 方法", func() {
			var user TestMongoUser
			err := record.ScanStruct(&user)
			So(err, ShouldBeNil)
			So(user.UserID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
		})
	})
}

func TestMongoRecordBuilder(t *testing.T) {
	Convey("测试 MongoRecordBuilder 方法", t, func() {
		builder := &MongoRecordBuilder{}

		Convey("测试 FromStruct 方法", func() {
			user := TestMongoUser{
				UserID:   1,
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      30,
				Active:   true,
				Score:    95.5,
				CreateAt: time.Now(),
			}

			record := builder.FromStruct(user)
			So(record, ShouldNotBeNil)

			fields := record.Fields()
			So(fields["user_id"], ShouldEqual, 1)
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
			So(fields["age"], ShouldEqual, 30)
			So(fields["active"], ShouldEqual, true)
			So(fields["score"], ShouldEqual, 95.5)
		})

		Convey("测试 FromMap 方法", func() {
			data := map[string]any{
				"user_id": 1,
				"name":    "John Doe",
				"email":   "john@example.com",
			}

			record := builder.FromMap(data, "users")
			So(record, ShouldNotBeNil)

			fields := record.Fields()
			So(fields["user_id"], ShouldEqual, 1)
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
		})
	})
}

func TestStructToBSON(t *testing.T) {
	Convey("测试 structToBSON 辅助函数", t, func() {
		Convey("正常结构体转换", func() {
			user := TestMongoUser{
				UserID: 1,
				Name:   "John Doe",
				Email:  "john@example.com",
				Age:    30,
			}

			result := structToBSON(user)
			So(result["user_id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "John Doe")
			So(result["email"], ShouldEqual, "john@example.com")
			So(result["age"], ShouldEqual, 30)
		})

		Convey("指针结构体转换", func() {
			user := &TestMongoUser{
				UserID: 1,
				Name:   "John Doe",
			}

			result := structToBSON(user)
			So(result["user_id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "John Doe")
		})

		Convey("非结构体类型", func() {
			result := structToBSON("not a struct")
			So(len(result), ShouldEqual, 0)
		})
	})
}

func TestBSONToStruct(t *testing.T) {
	Convey("测试 bsonToStruct 辅助函数", t, func() {
		Convey("正常转换", func() {
			data := map[string]any{
				"_id":       primitive.NewObjectID(),
				"user_id":   1,
				"name":      "John Doe",
				"email":     "john@example.com",
				"age":       30,
				"active":    true,
				"score":     95.5,
				"create_at": time.Now(),
			}

			var user TestMongoUser
			err := bsonToStruct(data, &user)
			So(err, ShouldBeNil)
			So(user.UserID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
		})

		Convey("目标不是指针", func() {
			data := map[string]any{"user_id": 1}
			var user TestMongoUser
			err := bsonToStruct(data, user)
			So(err, ShouldNotBeNil)
		})

		Convey("目标不是结构体指针", func() {
			data := map[string]any{"value": 1}
			var value int
			err := bsonToStruct(data, &value)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestMongoMigrate(t *testing.T) {
	Convey("测试 Mongo Migrate 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		Convey("创建带索引的集合", func() {
			model := &TableModel{
				Table: "test_users",
				Fields: []FieldDefinition{
					{Name: "_id", Type: FieldTypeString, Required: true},
					{Name: "user_id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
					{Name: "email", Type: FieldTypeString},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "create_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"_id"},
				Indexes: []IndexDefinition{
					{Name: "idx_email", Fields: []string{"email"}, Unique: true},
					{Name: "idx_user_id", Fields: []string{"user_id"}, Unique: true},
					{Name: "idx_name_age", Fields: []string{"name", "age"}},
				},
			}

			ctx := context.Background()
			err := mongo.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 清理测试集合
			mongo.DropTable(ctx, "test_users")
		})

		Convey("创建简单集合", func() {
			model := &TableModel{
				Table: "test_simple_collection",
				Fields: []FieldDefinition{
					{Name: "_id", Type: FieldTypeString, Required: true},
					{Name: "data", Type: FieldTypeJSON},
				},
				PrimaryKey: []string{"_id"},
			}

			ctx := context.Background()
			err := mongo.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 清理测试集合
			mongo.DropTable(ctx, "test_simple_collection")
		})
	})
}

func TestMongoCRUDOperations(t *testing.T) {
	Convey("测试 Mongo CRUD 操作", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		// 创建测试集合
		ctx := context.Background()
		model := &TableModel{
			Table: "test_crud_users",
			Fields: []FieldDefinition{
				{Name: "_id", Type: FieldTypeString, Required: true},
				{Name: "user_id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Required: true},
				{Name: "email", Type: FieldTypeString},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"_id"},
		}
		mongo.Migrate(ctx, model)
		defer mongo.DropTable(ctx, "test_crud_users")

		Convey("测试 Create 方法", func() {
			user := TestMongoUser{
				UserID:   1,
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      30,
				Active:   true,
				Score:    95.5,
				CreateAt: time.Now(),
			}

			record := mongo.builder.FromStruct(user)
			err := mongo.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)
		})

		Convey("测试 Create 方法的 IgnoreConflict 选项", func() {
			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   10,
				Name:     "Original User",
				Email:    "original@example.com",
				Age:      25,
				Active:   true,
				Score:    80.0,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			err := mongo.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)

			// 尝试创建相同_id的记录，使用 IgnoreConflict 选项
			conflictUser := TestMongoUser{
				ID:       objectID,
				UserID:   11,
				Name:     "Conflict User",
				Email:    "conflict@example.com",
				Age:      30,
				Active:   false,
				Score:    90.0,
				CreateAt: time.Now(),
			}
			conflictRecord := mongo.builder.FromStruct(conflictUser)

			// 使用 IgnoreConflict 选项，应该忽略冲突
			err = mongo.Create(ctx, "test_crud_users", conflictRecord, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证原始记录没有被修改
			pk := map[string]any{"_id": objectID}
			result, err := mongo.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestMongoUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Original User")
			So(retrievedUser.Email, ShouldEqual, "original@example.com")
		})

		Convey("测试 Create 方法的 UpdateOnConflict 选项", func() {
			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   12,
				Name:     "Original User",
				Email:    "original12@example.com",
				Age:      25,
				Active:   true,
				Score:    80.0,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			err := mongo.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)

			// 尝试创建相同_id的记录，使用 UpdateOnConflict 选项
			conflictUser := TestMongoUser{
				ID:       objectID,
				UserID:   13,
				Name:     "Updated User",
				Email:    "updated12@example.com",
				Age:      30,
				Active:   false,
				Score:    90.0,
				CreateAt: time.Now(),
			}
			conflictRecord := mongo.builder.FromStruct(conflictUser)

			// 使用 UpdateOnConflict 选项，应该更新记录
			err = mongo.Create(ctx, "test_crud_users", conflictRecord, WithUpdateOnConflict())
			So(err, ShouldBeNil)

			// 验证记录已被更新
			pk := map[string]any{"_id": objectID}
			result, err := mongo.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestMongoUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Updated User")
			So(retrievedUser.Email, ShouldEqual, "updated12@example.com")
			So(retrievedUser.Age, ShouldEqual, 30)
		})

		Convey("测试 Get 方法", func() {
			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   2,
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Age:      25,
				Active:   true,
				Score:    88.5,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			mongo.Create(ctx, "test_crud_users", record)

			// 获取记录
			pk := map[string]any{"_id": objectID}
			result, err := mongo.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			var retrievedUser TestMongoUser
			err = result.Scan(&retrievedUser)
			So(err, ShouldBeNil)
			So(retrievedUser.ID, ShouldEqual, objectID)
			So(retrievedUser.UserID, ShouldEqual, 2)
			So(retrievedUser.Name, ShouldEqual, "Jane Doe")
			So(retrievedUser.Email, ShouldEqual, "jane@example.com")
		})

		Convey("测试 Update 方法", func() {
			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   3,
				Name:     "Bob Smith",
				Email:    "bob@example.com",
				Age:      35,
				Active:   true,
				Score:    92.0,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			mongo.Create(ctx, "test_crud_users", record)

			// 更新记录
			updatedUser := TestMongoUser{
				ID:       objectID,
				UserID:   3,
				Name:     "Bob Smith Updated",
				Email:    "bob.updated@example.com",
				Age:      36,
				Active:   false,
				Score:    93.5,
				CreateAt: time.Now(),
			}
			updatedRecord := mongo.builder.FromStruct(updatedUser)
			pk := map[string]any{"_id": objectID}
			err := mongo.Update(ctx, "test_crud_users", pk, updatedRecord)
			So(err, ShouldBeNil)

			// 验证更新
			result, err := mongo.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestMongoUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Bob Smith Updated")
			So(retrievedUser.Email, ShouldEqual, "bob.updated@example.com")
			So(retrievedUser.Age, ShouldEqual, 36)
		})

		Convey("测试 Delete 方法", func() {
			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   4,
				Name:     "Alice Johnson",
				Email:    "alice@example.com",
				Age:      28,
				Active:   true,
				Score:    87.5,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			mongo.Create(ctx, "test_crud_users", record)

			// 删除记录
			pk := map[string]any{"_id": objectID}
			err := mongo.Delete(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)

			// 验证删除
			_, err = mongo.Get(ctx, "test_crud_users", pk)
			So(err, ShouldEqual, ErrRecordNotFound)
		})
	})
}

func TestMongoFind(t *testing.T) {
	Convey("测试 Mongo Find 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		// 创建测试集合和数据
		ctx := context.Background()
		
		// 使用动态表名避免冲突
		tableName := fmt.Sprintf("test_find_users_%d", time.Now().UnixNano())
		defer mongo.DropTable(ctx, tableName)

		// 插入测试数据
		users := []TestMongoUser{
			{UserID: 1, Name: "John", Age: 30, Active: true, CreateAt: time.Now()},
			{UserID: 2, Name: "Jane", Age: 25, Active: true, CreateAt: time.Now()},
			{UserID: 3, Name: "Bob", Age: 35, Active: false, CreateAt: time.Now()},
			{UserID: 4, Name: "Alice", Age: 28, Active: true, CreateAt: time.Now()},
		}
		for _, user := range users {
			record := mongo.builder.FromStruct(user)
			err := mongo.Create(ctx, tableName, record)
			So(err, ShouldBeNil)
		}

		Convey("使用 TermQuery 查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			results, err := mongo.Find(ctx, tableName, termQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3) // John, Jane, Alice
		})

		Convey("使用 MatchQuery 查询", func() {
			matchQuery := &query.MatchQuery{Field: "name", Value: "Jo"}
			results, err := mongo.Find(ctx, tableName, matchQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1) // John
		})

		Convey("带排序的查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			options := &QueryOptions{OrderBy: "age", OrderDesc: false}
			results, err := mongo.Find(ctx, tableName, termQuery, func(opts *QueryOptions) {
				*opts = *options
			})
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)

			// 验证排序 (Jane:25, Alice:28, John:30)
			var firstUser TestMongoUser
			results[0].Scan(&firstUser)
			So(firstUser.Age, ShouldEqual, 25) // Jane
		})

		Convey("带分页的查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			options := &QueryOptions{Limit: 2, Offset: 1}
			results, err := mongo.Find(ctx, tableName, termQuery, func(opts *QueryOptions) {
				*opts = *options
			})
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)
		})
	})
}

func TestMongoAggregate(t *testing.T) {
	Convey("测试 Mongo Aggregate 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		// 创建测试集合和数据
		ctx := context.Background()
		model := &TableModel{
			Table: "test_agg_users",
			Fields: []FieldDefinition{
				{Name: "_id", Type: FieldTypeString, Required: true},
				{Name: "user_id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "active", Type: FieldTypeBool, Default: true},
			},
			PrimaryKey: []string{"_id"},
		}
		mongo.Migrate(ctx, model)
		defer mongo.DropTable(ctx, "test_agg_users")

		// 插入测试数据，包含一些空值用于测试COUNT(field)
		users := []TestMongoUser{
			{UserID: 1, Name: "John", Email: "john@test.com", Age: 30, Score: 95.5, Active: true},
			{UserID: 2, Name: "Jane", Email: "", Age: 25, Score: 88.0, Active: true}, // 空email
			{UserID: 3, Name: "Bob", Email: "bob@test.com", Age: 35, Score: 92.5, Active: false},
			{UserID: 4, Name: "", Email: "alice@test.com", Age: 28, Score: 0, Active: true}, // 空name，score为0
			{UserID: 5, Name: "Charlie", Email: "charlie@test.com", Age: 32, Score: 90.0, Active: true},
		}
		for _, user := range users {
			record := mongo.builder.FromStruct(user)
			mongo.Create(ctx, "test_agg_users", record)
		}

		Convey("Count 聚合 - COUNT(*)", func() {
			// 测试 COUNT(*) - 统计所有active=true的用户
			termQuery := &query.TermQuery{Field: "active", Value: true}
			countAgg := &aggregation.CountAggregation{}
			countAgg.AggName = "total_count"
			countAgg.Field = "" // 空字段表示COUNT(*)

			aggs := []aggregation.Aggregation{countAgg}
			result, err := mongo.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			
			// 验证结果：应该有4个active=true的用户
			count := result.Get("total_count")
			So(count, ShouldEqual, 4)
		})

		Convey("Count 聚合 - COUNT(email)", func() {
			// 测试 COUNT(email) - 统计有非空email的active用户
			termQuery := &query.TermQuery{Field: "active", Value: true}
			countAgg := &aggregation.CountAggregation{}
			countAgg.AggName = "email_count"
			countAgg.Field = "email"

			aggs := []aggregation.Aggregation{countAgg}
			result, err := mongo.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			
			// 验证结果：4个active用户中，3个有非空email（Jane的email为空）
			count := result.Get("email_count")
			So(count, ShouldEqual, 3)
		})

		Convey("Count 聚合 - COUNT(name)", func() {
			// 测试 COUNT(name) - 统计有非空name的active用户
			termQuery := &query.TermQuery{Field: "active", Value: true}
			countAgg := &aggregation.CountAggregation{}
			countAgg.AggName = "name_count"
			countAgg.Field = "name"

			aggs := []aggregation.Aggregation{countAgg}
			result, err := mongo.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			
			// 验证结果：4个active用户中，3个有非空name（用户4的name为空）
			count := result.Get("name_count")
			So(count, ShouldEqual, 3)
		})

		Convey("Count 聚合 - COUNT(score)", func() {
			// 测试 COUNT(score) - 统计有score的active用户（数字0也算有效值）
			termQuery := &query.TermQuery{Field: "active", Value: true}
			countAgg := &aggregation.CountAggregation{}
			countAgg.AggName = "score_count"
			countAgg.Field = "score"

			aggs := []aggregation.Aggregation{countAgg}
			result, err := mongo.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			
			// 验证结果：所有4个active用户都有score（包括用户4的score=0）
			count := result.Get("score_count")
			So(count, ShouldEqual, 4)
		})

		Convey("Count 聚合 - 多字段COUNT组合", func() {
			// 测试在同一个聚合中使用多个COUNT
			termQuery := &query.TermQuery{Field: "active", Value: true}
			
			// 同时统计多个字段的count
			countAllAgg := &aggregation.CountAggregation{}
			countAllAgg.AggName = "total_count"
			countAllAgg.Field = ""
			
			countEmailAgg := &aggregation.CountAggregation{}
			countEmailAgg.AggName = "email_count"
			countEmailAgg.Field = "email"
			
			countNameAgg := &aggregation.CountAggregation{}
			countNameAgg.AggName = "name_count"
			countNameAgg.Field = "name"

			aggs := []aggregation.Aggregation{countAllAgg, countEmailAgg, countNameAgg}
			result, err := mongo.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			
			// 验证多个COUNT结果
			totalCount := result.Get("total_count")
			emailCount := result.Get("email_count")
			nameCount := result.Get("name_count")
			
			So(totalCount, ShouldEqual, 4) // 总active用户数
			So(emailCount, ShouldEqual, 3) // 有email的active用户数
			So(nameCount, ShouldEqual, 3)  // 有name的active用户数
		})
	})
}

func TestMongoBatchOperations(t *testing.T) {
	Convey("测试 Mongo 批量操作", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		// 创建测试集合
		ctx := context.Background()
		model := &TableModel{
			Table: "test_batch_users",
			Fields: []FieldDefinition{
				{Name: "_id", Type: FieldTypeString, Required: true},
				{Name: "user_id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "email", Type: FieldTypeString},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"_id"},
		}
		mongo.Migrate(ctx, model)
		defer mongo.DropTable(ctx, "test_batch_users")

		Convey("测试 BatchCreate", func() {
			users := []TestMongoUser{
				{UserID: 1, Name: "User1", Age: 20, CreateAt: time.Now()},
				{UserID: 2, Name: "User2", Age: 21, CreateAt: time.Now()},
				{UserID: 3, Name: "User3", Age: 22, CreateAt: time.Now()},
			}

			var records []Record
			for _, user := range users {
				records = append(records, mongo.builder.FromStruct(user))
			}

			err := mongo.BatchCreate(ctx, "test_batch_users", records)
			So(err, ShouldBeNil)
		})

		Convey("测试 BatchCreate 的 CreateOption", func() {
			// 先创建一些记录
			objectID1 := primitive.NewObjectID()
			objectID2 := primitive.NewObjectID()
			users := []TestMongoUser{
				{ID: objectID1, UserID: 50, Name: "OriginalUser50", Age: 20, CreateAt: time.Now()},
				{ID: objectID2, UserID: 51, Name: "OriginalUser51", Age: 21, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, mongo.builder.FromStruct(user))
			}
			err := mongo.BatchCreate(ctx, "test_batch_users", records)
			So(err, ShouldBeNil)

			// 测试批量创建时使用 IgnoreConflict 选项
			conflictUsers := []TestMongoUser{
				{ID: objectID1, UserID: 50, Name: "ConflictUser50", Age: 30, CreateAt: time.Now()}, // 冲突记录
				{UserID: 52, Name: "NewUser52", Age: 22, CreateAt: time.Now()},                     // 新记录
			}
			var conflictRecords []Record
			for _, user := range conflictUsers {
				conflictRecords = append(conflictRecords, mongo.builder.FromStruct(user))
			}

			err = mongo.BatchCreate(ctx, "test_batch_users", conflictRecords, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证ID=objectID1的记录没有被修改
			pk50 := map[string]any{"_id": objectID1}
			result50, err := mongo.Get(ctx, "test_batch_users", pk50)
			So(err, ShouldBeNil)
			var user50 TestMongoUser
			result50.Scan(&user50)
			So(user50.Name, ShouldEqual, "OriginalUser50") // 原始记录没有被修改
		})

		Convey("测试 BatchUpdate", func() {
			// 先创建记录
			objectID1 := primitive.NewObjectID()
			objectID2 := primitive.NewObjectID()
			users := []TestMongoUser{
				{ID: objectID1, UserID: 4, Name: "User4", Age: 23, CreateAt: time.Now()},
				{ID: objectID2, UserID: 5, Name: "User5", Age: 24, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, mongo.builder.FromStruct(user))
			}
			mongo.BatchCreate(ctx, "test_batch_users", records)

			// 批量更新
			updatedUsers := []TestMongoUser{
				{ID: objectID1, UserID: 4, Name: "Updated User4", Age: 33, CreateAt: time.Now()},
				{ID: objectID2, UserID: 5, Name: "Updated User5", Age: 34, CreateAt: time.Now()},
			}
			var updatedRecords []Record
			var pks []map[string]any
			for _, user := range updatedUsers {
				updatedRecords = append(updatedRecords, mongo.builder.FromStruct(user))
				pks = append(pks, map[string]any{"_id": user.ID})
			}

			err := mongo.BatchUpdate(ctx, "test_batch_users", pks, updatedRecords)
			So(err, ShouldBeNil)
		})

		Convey("测试 BatchDelete", func() {
			// 先创建记录
			objectID1 := primitive.NewObjectID()
			objectID2 := primitive.NewObjectID()
			users := []TestMongoUser{
				{ID: objectID1, UserID: 6, Name: "User6", Age: 25, CreateAt: time.Now()},
				{ID: objectID2, UserID: 7, Name: "User7", Age: 26, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, mongo.builder.FromStruct(user))
			}
			mongo.BatchCreate(ctx, "test_batch_users", records)

			// 批量删除
			pks := []map[string]any{
				{"_id": objectID1},
				{"_id": objectID2},
			}

			err := mongo.BatchDelete(ctx, "test_batch_users", pks)
			So(err, ShouldBeNil)
		})
	})
}

func TestMongoTransaction(t *testing.T) {
	Convey("测试 Mongo 事务操作", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		// 创建测试集合
		ctx := context.Background()
		
		// 检查MongoDB是否支持事务（需要副本集或分片集群）
		// 尝试创建并提交一个简单事务来检测支持情况
		testTx, err := mongo.BeginTx(ctx)
		if err == nil && testTx != nil {
			// 尝试执行一个简单的事务操作来检测是否真正支持事务
			testRecord := mongo.GetBuilder().FromMap(map[string]any{"test": "value"}, "test_table")
			err = testTx.Create(ctx, "test_transaction_check", testRecord)
			testTx.Rollback() // 清理测试事务
		}
		
		if err != nil && (strings.Contains(err.Error(), "Transaction numbers are only allowed") || 
						strings.Contains(err.Error(), "replica set")) {
			SkipConvey("跳过事务测试：MongoDB实例不支持事务（需要副本集配置）", func() {})
			return
		}
		model := &TableModel{
			Table: "test_tx_users",
			Fields: []FieldDefinition{
				{Name: "_id", Type: FieldTypeString, Required: true},
				{Name: "user_id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "email", Type: FieldTypeString},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"_id"},
		}
		mongo.Migrate(ctx, model)
		defer mongo.DropTable(ctx, "test_tx_users")

		Convey("测试 BeginTx 和手动提交", func() {
			tx, err := mongo.BeginTx(ctx)
			So(err, ShouldBeNil)
			So(tx, ShouldNotBeNil)

			user := TestMongoUser{UserID: 1, Name: "TxUser1", Age: 30, CreateAt: time.Now()}
			record := mongo.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			err = tx.Commit()
			So(err, ShouldBeNil)

			// 验证提交成功（注意：需要通过其他字段查询，因为_id是自动生成的）
			termQuery := &query.TermQuery{Field: "user_id", Value: 1}
			results, err := mongo.Find(ctx, "test_tx_users", termQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldBeGreaterThan, 0)
		})

		Convey("测试事务回滚", func() {
			tx, err := mongo.BeginTx(ctx)
			So(err, ShouldBeNil)

			user := TestMongoUser{UserID: 2, Name: "TxUser2", Age: 25, CreateAt: time.Now()}
			record := mongo.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			err = tx.Rollback()
			So(err, ShouldBeNil)

			// 验证回滚成功
			termQuery := &query.TermQuery{Field: "user_id", Value: 2}
			results, err := mongo.Find(ctx, "test_tx_users", termQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 0)
		})

		Convey("测试 WithTx", func() {
			err := mongo.WithTx(ctx, func(tx Transaction) error {
				user := TestMongoUser{UserID: 3, Name: "TxUser3", Age: 28, CreateAt: time.Now()}
				record := mongo.builder.FromStruct(user)
				return tx.Create(ctx, "test_tx_users", record)
			})
			So(err, ShouldBeNil)

			// 验证提交成功
			termQuery := &query.TermQuery{Field: "user_id", Value: 3}
			results, err := mongo.Find(ctx, "test_tx_users", termQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldBeGreaterThan, 0)
		})

		Convey("测试事务中的 CreateOption", func() {
			tx, err := mongo.BeginTx(ctx)
			So(err, ShouldBeNil)
			defer tx.Rollback()

			// 先创建一条记录
			objectID := primitive.NewObjectID()
			user := TestMongoUser{
				ID:       objectID,
				UserID:   100,
				Name:     "TxOriginal",
				Email:    "txoriginal@example.com",
				Age:      25,
				Active:   true,
				Score:    85.0,
				CreateAt: time.Now(),
			}
			record := mongo.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			// 测试 IgnoreConflict 选项
			conflictUser := TestMongoUser{
				ID:       objectID,
				UserID:   101,
				Name:     "TxConflict",
				Email:    "txconflict@example.com",
				Age:      30,
				Active:   false,
				Score:    95.0,
				CreateAt: time.Now(),
			}
			conflictRecord := mongo.builder.FromStruct(conflictUser)

			err = tx.Create(ctx, "test_tx_users", conflictRecord, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证原始记录没有被修改（在事务中）
			pk := map[string]any{"_id": objectID}
			result, err := tx.Get(ctx, "test_tx_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestMongoUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "TxOriginal")
			So(retrievedUser.Email, ShouldEqual, "txoriginal@example.com")

			// 测试 UpdateOnConflict 选项
			updateUser := TestMongoUser{
				ID:       objectID,
				UserID:   102,
				Name:     "TxUpdated",
				Email:    "txupdated@example.com",
				Age:      35,
				Active:   false,
				Score:    99.0,
				CreateAt: time.Now(),
			}
			updateRecord := mongo.builder.FromStruct(updateUser)

			err = tx.Create(ctx, "test_tx_users", updateRecord, WithUpdateOnConflict())
			So(err, ShouldBeNil)

			// 验证记录已被更新（在事务中）
			result, err = tx.Get(ctx, "test_tx_users", pk)
			So(err, ShouldBeNil)
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "TxUpdated")
			So(retrievedUser.Email, ShouldEqual, "txupdated@example.com")
			So(retrievedUser.Age, ShouldEqual, 35)
		})
	})
}

func TestMongoGetBuilder(t *testing.T) {
	Convey("测试 Mongo GetBuilder 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		builder := mongo.GetBuilder()
		So(builder, ShouldNotBeNil)
		So(builder, ShouldEqual, mongo.builder)
	})
}

func TestMongoClose(t *testing.T) {
	Convey("测试 Mongo Close 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)

		err = mongo.Close()
		So(err, ShouldBeNil)
	})
}

func TestMongoDropTable(t *testing.T) {
	Convey("测试 Mongo DropTable 方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		ctx := context.Background()

		Convey("删除存在的集合", func() {
			// 先创建一个测试集合
			model := &TableModel{
				Table: "test_drop_collection_exists",
				Fields: []FieldDefinition{
					{Name: "_id", Type: FieldTypeString, Required: true},
					{Name: "user_id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Required: true},
				},
				PrimaryKey: []string{"_id"},
			}
			err := mongo.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 删除集合
			err = mongo.DropTable(ctx, "test_drop_collection_exists")
			So(err, ShouldBeNil)
		})

		Convey("删除不存在的集合", func() {
			// 删除不存在的集合应该不会报错
			err := mongo.DropTable(ctx, "test_drop_collection_not_exists")
			So(err, ShouldBeNil)
		})
	})
}

func TestMongoErrorHandling(t *testing.T) {
	Convey("测试 Mongo 错误处理", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		ctx := context.Background()

		Convey("测试获取不存在的记录", func() {
			pk := map[string]any{"_id": primitive.NewObjectID()}
			_, err := mongo.Get(ctx, "non_existent_collection", pk)
			So(err, ShouldEqual, ErrRecordNotFound)
		})

		Convey("测试 bsonToStruct 错误情况", func() {
			data := map[string]any{"user_id": "not_a_number"}
			var user TestMongoUser
			// 这个测试可能会成功，因为 Go 的反射会尝试类型转换
			// 但我们至少验证了函数不会 panic
			So(func() { bsonToStruct(data, &user) }, ShouldNotPanic)
		})
	})
}

func TestMongoEdgeCases(t *testing.T) {
	Convey("测试 Mongo 边界情况", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		Convey("测试空字段的结构体", func() {
			type EmptyStruct struct{}
			empty := EmptyStruct{}
			result := structToBSON(empty)
			So(len(result), ShouldEqual, 0)
		})

		Convey("测试带有未导出字段的结构体", func() {
			type StructWithPrivateFields struct {
				UserID       int    `bson:"user_id" rdb:"user_id"`
				privateField string // 未导出字段
				Name         string `bson:"name" rdb:"name"`
			}

			s := StructWithPrivateFields{
				UserID:       1,
				privateField: "private",
				Name:         "test",
			}

			result := structToBSON(s)
			So(result["user_id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "test")
			So(result["privateField"], ShouldBeNil) // 未导出字段不应该被包含
		})

		Convey("测试带有 rdb:'-' 标签的字段", func() {
			type StructWithIgnoredField struct {
				UserID  int    `bson:"user_id" rdb:"user_id"`
				Ignored string `bson:"ignored" rdb:"-"`
				Name    string `bson:"name" rdb:"name"`
			}

			s := StructWithIgnoredField{
				UserID:  1,
				Ignored: "ignored",
				Name:    "test",
			}

			result := structToBSON(s)
			So(result["user_id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "test")
			_, exists := result["Ignored"]
			So(exists, ShouldBeFalse) // 被忽略的字段不应该被包含
		})

		Convey("测试批量操作长度不匹配", func() {
			pks := []map[string]any{{"_id": primitive.NewObjectID()}, {"_id": primitive.NewObjectID()}}
			records := []Record{mongo.builder.FromStruct(TestMongoUser{UserID: 1})} // 只有一个记录

			ctx := context.Background()
			err := mongo.BatchUpdate(ctx, "test_collection", pks, records)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "length mismatch")
		})
	})
}

func TestMongoTransactionMethods(t *testing.T) {
	Convey("测试 MongoTransaction 特有方法", t, func() {
		mongo, err := NewMongoWithOptions(testMongoOptions)
		So(err, ShouldBeNil)
		defer mongo.Close()

		ctx := context.Background()
		
		// 检查MongoDB是否支持事务
		tx, err := mongo.BeginTx(ctx)
		if err != nil && (strings.Contains(err.Error(), "Transaction numbers are only allowed") || 
						strings.Contains(err.Error(), "replica set")) {
			SkipConvey("跳过事务方法测试：MongoDB实例不支持事务（需要副本集配置）", func() {})
			return
		}
		So(err, ShouldBeNil)
		defer tx.Rollback()

		Convey("测试事务的 GetBuilder", func() {
			builder := tx.GetBuilder()
			So(builder, ShouldNotBeNil)
		})

		Convey("测试事务的 Close", func() {
			err := tx.Close()
			So(err, ShouldBeNil) // 事务的 Close 应该返回 nil
		})

		Convey("测试嵌套事务", func() {
			nestedTx, err := tx.BeginTx(ctx)
			So(err, ShouldNotBeNil)
			So(nestedTx, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "nested transactions not supported")
		})

		Convey("测试事务的 WithTx", func() {
			err := tx.WithTx(ctx, func(innerTx Transaction) error {
				So(innerTx, ShouldEqual, tx)
				return nil
			})
			So(err, ShouldBeNil)
		})

		Convey("测试事务的 Migrate", func() {
			model := &TableModel{
				Table: "test_tx_migrate",
				Fields: []FieldDefinition{
					{Name: "_id", Type: FieldTypeString, Required: true},
					{Name: "name", Type: FieldTypeString},
				},
				PrimaryKey: []string{"_id"},
			}

			err := tx.Migrate(ctx, model)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "schema migration not supported in transactions")
		})

		Convey("测试事务的 DropTable", func() {
			err := tx.DropTable(ctx, "test_tx_drop")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "drop table not supported in transactions")
		})
	})
}