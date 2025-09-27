package database

import (
	"context"
	"testing"
	"time"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的结构体
type TestUser struct {
	ID       int       `rdb:"id"`
	Name     string    `rdb:"name"`
	Email    string    `rdb:"email"`
	Age      int       `rdb:"age"`
	Active   bool      `rdb:"active"`
	Score    float64   `rdb:"score"`
	CreateAt time.Time `rdb:"create_at"`
}

// 测试配置
var testMySQLOptions = &SQLOptions{
	Driver:   "mysql",
	Host:     "localhost",
	Port:     "3306",
	Database: "testdb",
	Username: "testuser",
	Password: "testpass",
	Charset:  "utf8mb4",
	MaxConns: 10,
	MaxIdle:  5,
}

func TestNewSQLWithOptions(t *testing.T) {
	Convey("测试 NewSQLWithOptions 方法", t, func() {
		Convey("使用 MySQL 驱动创建连接", func() {
			sql, err := NewSQLWithOptions(testMySQLOptions)
			So(err, ShouldBeNil)
			So(sql, ShouldNotBeNil)
			So(sql.driver, ShouldEqual, "mysql")
			So(sql.db, ShouldNotBeNil)
			So(sql.builder, ShouldNotBeNil)

			// 清理资源
			sql.Close()
		})

		Convey("使用自定义 DSN", func() {
			options := &SQLOptions{
				Driver: "mysql",
				DSN:    "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
			}
			sql, err := NewSQLWithOptions(options)
			So(err, ShouldBeNil)
			So(sql, ShouldNotBeNil)

			// 清理资源
			sql.Close()
		})

		Convey("使用 SQLite3 驱动", func() {
			options := &SQLOptions{
				Driver:   "sqlite3",
				Database: ":memory:",
			}
			sql, err := NewSQLWithOptions(options)
			So(err, ShouldBeNil)
			So(sql, ShouldNotBeNil)
			So(sql.driver, ShouldEqual, "sqlite3")

			// 清理资源
			sql.Close()
		})

		Convey("不支持的驱动类型", func() {
			options := &SQLOptions{
				Driver: "unsupported",
			}
			sql, err := NewSQLWithOptions(options)
			So(err, ShouldNotBeNil)
			So(sql, ShouldBeNil)
		})
	})
}

func TestSQLRecord(t *testing.T) {
	Convey("测试 SQLRecord 方法", t, func() {
		data := map[string]any{
			"id":        1,
			"name":      "John Doe",
			"email":     "john@example.com",
			"age":       30,
			"active":    true,
			"score":     95.5,
			"create_at": time.Now(),
		}
		record := &SQLRecord{data: data}

		Convey("测试 Fields 方法", func() {
			fields := record.Fields()
			So(fields, ShouldResemble, data)
		})

		Convey("测试 Scan 方法", func() {
			var user TestUser
			err := record.Scan(&user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
		})

		Convey("测试 ScanStruct 方法", func() {
			var user TestUser
			err := record.ScanStruct(&user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
		})
	})
}

func TestSQLRecordBuilder(t *testing.T) {
	Convey("测试 SQLRecordBuilder 方法", t, func() {
		builder := &SQLRecordBuilder{}

		Convey("测试 FromStruct 方法", func() {
			user := TestUser{
				ID:       1,
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
			So(fields["id"], ShouldEqual, 1)
			So(fields["name"], ShouldEqual, "John Doe")
			So(fields["email"], ShouldEqual, "john@example.com")
			So(fields["age"], ShouldEqual, 30)
			So(fields["active"], ShouldEqual, true)
			So(fields["score"], ShouldEqual, 95.5)
		})

		Convey("测试 FromMap 方法", func() {
			data := map[string]any{
				"id":    1,
				"name":  "John Doe",
				"email": "john@example.com",
			}

			record := builder.FromMap(data, "users")
			So(record, ShouldNotBeNil)

			fields := record.Fields()
			So(fields, ShouldResemble, data)
		})
	})
}

func TestStructToMap(t *testing.T) {
	Convey("测试 structToMap 辅助函数", t, func() {
		Convey("正常结构体转换", func() {
			user := TestUser{
				ID:    1,
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			}

			result := structToMap(user)
			So(result["id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "John Doe")
			So(result["email"], ShouldEqual, "john@example.com")
			So(result["age"], ShouldEqual, 30)
		})

		Convey("指针结构体转换", func() {
			user := &TestUser{
				ID:   1,
				Name: "John Doe",
			}

			result := structToMap(user)
			So(result["id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "John Doe")
		})

		Convey("非结构体类型", func() {
			result := structToMap("not a struct")
			So(len(result), ShouldEqual, 0)
		})
	})
}

func TestMapToStruct(t *testing.T) {
	Convey("测试 mapToStruct 辅助函数", t, func() {
		Convey("正常转换", func() {
			data := map[string]any{
				"id":     1,
				"name":   "John Doe",
				"email":  "john@example.com",
				"age":    30,
				"active": true,
				"score":  95.5,
			}

			var user TestUser
			err := mapToStruct(data, &user)
			So(err, ShouldBeNil)
			So(user.ID, ShouldEqual, 1)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
			So(user.Active, ShouldEqual, true)
			So(user.Score, ShouldEqual, 95.5)
		})

		Convey("目标不是指针", func() {
			data := map[string]any{"id": 1}
			var user TestUser
			err := mapToStruct(data, user)
			So(err, ShouldNotBeNil)
		})

		Convey("目标不是结构体指针", func() {
			data := map[string]any{"value": 1}
			var value int
			err := mapToStruct(data, &value)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSQLMigrate(t *testing.T) {
	Convey("测试 SQL Migrate 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		Convey("创建简单表", func() {
			model := &TableModel{
				Table: "test_users",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
					{Name: "email", Type: FieldTypeString, Size: 255},
					{Name: "age", Type: FieldTypeInt},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "created_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
				Indexes: []IndexDefinition{
					{Name: "idx_email", Fields: []string{"email"}, Unique: true},
					{Name: "idx_name_age", Fields: []string{"name", "age"}},
				},
			}

			ctx := context.Background()
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 清理测试表
			sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_users")
		})

		Convey("创建带 JSON 字段的表", func() {
			model := &TableModel{
				Table: "test_json_table",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "data", Type: FieldTypeJSON},
				},
				PrimaryKey: []string{"id"},
			}

			ctx := context.Background()
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 清理测试表
			sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_json_table")
		})
	})
}

func TestSQLCRUDOperations(t *testing.T) {
	Convey("测试 SQL CRUD 操作", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		// 创建测试表
		ctx := context.Background()
		model := &TableModel{
			Table: "test_crud_users",
			Fields: []FieldDefinition{
				{Name: "id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				{Name: "email", Type: FieldTypeString, Size: 255},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"id"},
		}
		sql.Migrate(ctx, model)
		defer sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_crud_users")

		Convey("测试 Create 方法", func() {
			user := TestUser{
				ID:       1,
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      30,
				Active:   true,
				Score:    95.5,
				CreateAt: time.Now(),
			}

			record := sql.builder.FromStruct(user)
			err := sql.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)
		})

		Convey("测试 Create 方法的 IgnoreConflict 选项", func() {
			// 先创建一条记录
			user := TestUser{
				ID:       10,
				Name:     "Original User",
				Email:    "original@example.com",
				Age:      25,
				Active:   true,
				Score:    80.0,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			err := sql.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)

			// 尝试创建相同ID的记录，使用 IgnoreConflict 选项
			conflictUser := TestUser{
				ID:       10,
				Name:     "Conflict User",
				Email:    "conflict@example.com",
				Age:      30,
				Active:   false,
				Score:    90.0,
				CreateAt: time.Now(),
			}
			conflictRecord := sql.builder.FromStruct(conflictUser)
			
			// 使用 IgnoreConflict 选项，应该忽略冲突
			err = sql.Create(ctx, "test_crud_users", conflictRecord, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证原始记录没有被修改
			pk := map[string]any{"id": 10}
			result, err := sql.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Original User")
			So(retrievedUser.Email, ShouldEqual, "original@example.com")
		})

		Convey("测试 Create 方法的 UpdateOnConflict 选项", func() {
			// 先创建一条记录
			user := TestUser{
				ID:       11,
				Name:     "Original User",
				Email:    "original11@example.com",
				Age:      25,
				Active:   true,
				Score:    80.0,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			err := sql.Create(ctx, "test_crud_users", record)
			So(err, ShouldBeNil)

			// 尝试创建相同ID的记录，使用 UpdateOnConflict 选项
			conflictUser := TestUser{
				ID:       11,
				Name:     "Updated User",
				Email:    "updated11@example.com",
				Age:      30,
				Active:   false,
				Score:    90.0,
				CreateAt: time.Now(),
			}
			conflictRecord := sql.builder.FromStruct(conflictUser)
			
			// 使用 UpdateOnConflict 选项，应该更新记录
			err = sql.Create(ctx, "test_crud_users", conflictRecord, WithUpdateOnConflict())
			So(err, ShouldBeNil)

			// 验证记录已被更新
			pk := map[string]any{"id": 11}
			result, err := sql.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Updated User")
			So(retrievedUser.Email, ShouldEqual, "updated11@example.com")
			So(retrievedUser.Age, ShouldEqual, 30)
		})

		Convey("测试 Get 方法", func() {
			// 先创建一条记录
			user := TestUser{
				ID:       2,
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Age:      25,
				Active:   true,
				Score:    88.5,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			sql.Create(ctx, "test_crud_users", record)

			// 获取记录
			pk := map[string]any{"id": 2}
			result, err := sql.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			var retrievedUser TestUser
			err = result.Scan(&retrievedUser)
			So(err, ShouldBeNil)
			So(retrievedUser.ID, ShouldEqual, 2)
			So(retrievedUser.Name, ShouldEqual, "Jane Doe")
			So(retrievedUser.Email, ShouldEqual, "jane@example.com")
		})

		Convey("测试 Update 方法", func() {
			// 先创建一条记录
			user := TestUser{
				ID:       3,
				Name:     "Bob Smith",
				Email:    "bob@example.com",
				Age:      35,
				Active:   true,
				Score:    92.0,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			sql.Create(ctx, "test_crud_users", record)

			// 更新记录
			updatedUser := TestUser{
				ID:       3,
				Name:     "Bob Smith Updated",
				Email:    "bob.updated@example.com",
				Age:      36,
				Active:   false,
				Score:    93.5,
				CreateAt: time.Now(),
			}
			updatedRecord := sql.builder.FromStruct(updatedUser)
			pk := map[string]any{"id": 3}
			err := sql.Update(ctx, "test_crud_users", pk, updatedRecord)
			So(err, ShouldBeNil)

			// 验证更新
			result, err := sql.Get(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "Bob Smith Updated")
			So(retrievedUser.Email, ShouldEqual, "bob.updated@example.com")
			So(retrievedUser.Age, ShouldEqual, 36)
		})

		Convey("测试 Delete 方法", func() {
			// 先创建一条记录
			user := TestUser{
				ID:       4,
				Name:     "Alice Johnson",
				Email:    "alice@example.com",
				Age:      28,
				Active:   true,
				Score:    87.5,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			sql.Create(ctx, "test_crud_users", record)

			// 删除记录
			pk := map[string]any{"id": 4}
			err := sql.Delete(ctx, "test_crud_users", pk)
			So(err, ShouldBeNil)

			// 验证删除
			_, err = sql.Get(ctx, "test_crud_users", pk)
			So(err, ShouldEqual, ErrRecordNotFound)
		})
	})
}

func TestSQLFind(t *testing.T) {
	Convey("测试 SQL Find 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		// 创建测试表和数据
		ctx := context.Background()
		model := &TableModel{
			Table: "test_find_users",
			Fields: []FieldDefinition{
				{Name: "id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "email", Type: FieldTypeString, Size: 255},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"id"},
		}
		sql.Migrate(ctx, model)
		defer sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_find_users")

		// 插入测试数据
		users := []TestUser{
			{ID: 1, Name: "John", Age: 30, Active: true, CreateAt: time.Now()},
			{ID: 2, Name: "Jane", Age: 25, Active: true, CreateAt: time.Now()},
			{ID: 3, Name: "Bob", Age: 35, Active: false, CreateAt: time.Now()},
			{ID: 4, Name: "Alice", Age: 28, Active: true, CreateAt: time.Now()},
		}
		for _, user := range users {
			record := sql.builder.FromStruct(user)
			sql.Create(ctx, "test_find_users", record)
		}

		Convey("使用 TermQuery 查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			results, err := sql.Find(ctx, "test_find_users", termQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3) // John, Jane, Alice
		})

		Convey("使用 MatchQuery 查询", func() {
			matchQuery := &query.MatchQuery{Field: "name", Value: "Jo"}
			results, err := sql.Find(ctx, "test_find_users", matchQuery)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1) // John
		})

		Convey("带排序的查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			options := &QueryOptions{OrderBy: "age", OrderDesc: false}
			results, err := sql.Find(ctx, "test_find_users", termQuery, func(opts *QueryOptions) {
				*opts = *options
			})
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)

			// 验证排序 (Jane:25, Alice:28, John:30)
			var firstUser TestUser
			results[0].Scan(&firstUser)
			So(firstUser.Age, ShouldEqual, 25) // Jane
		})

		Convey("带分页的查询", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			options := &QueryOptions{Limit: 2, Offset: 1}
			results, err := sql.Find(ctx, "test_find_users", termQuery, func(opts *QueryOptions) {
				*opts = *options
			})
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)
		})
	})
}

func TestSQLAggregate(t *testing.T) {
	Convey("测试 SQL Aggregate 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		// 创建测试表和数据
		ctx := context.Background()
		model := &TableModel{
			Table: "test_agg_users",
			Fields: []FieldDefinition{
				{Name: "id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "active", Type: FieldTypeBool, Default: true},
			},
			PrimaryKey: []string{"id"},
		}
		sql.Migrate(ctx, model)
		defer sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_agg_users")

		// 插入测试数据
		users := []TestUser{
			{ID: 1, Name: "John", Age: 30, Score: 95.5, Active: true},
			{ID: 2, Name: "Jane", Age: 25, Score: 88.0, Active: true},
			{ID: 3, Name: "Bob", Age: 35, Score: 92.5, Active: false},
		}
		for _, user := range users {
			record := sql.builder.FromStruct(user)
			sql.Create(ctx, "test_agg_users", record)
		}

		Convey("Count 聚合", func() {
			termQuery := &query.TermQuery{Field: "active", Value: true}
			countAgg := &aggregation.CountAggregation{}
			countAgg.AggName = "total_count"
			countAgg.Field = "id"

			aggs := []aggregation.Aggregation{countAgg}
			result, err := sql.Aggregate(ctx, "test_agg_users", termQuery, aggs)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})
	})
}

func TestSQLBatchOperations(t *testing.T) {
	Convey("测试 SQL 批量操作", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		// 创建测试表
		ctx := context.Background()
		model := &TableModel{
			Table: "test_batch_users",
			Fields: []FieldDefinition{
				{Name: "id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "email", Type: FieldTypeString, Size: 255},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"id"},
		}
		sql.Migrate(ctx, model)
		defer sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_batch_users")

		Convey("测试 BatchCreate", func() {
			users := []TestUser{
				{ID: 1, Name: "User1", Age: 20, CreateAt: time.Now()},
				{ID: 2, Name: "User2", Age: 21, CreateAt: time.Now()},
				{ID: 3, Name: "User3", Age: 22, CreateAt: time.Now()},
			}

			var records []Record
			for _, user := range users {
				records = append(records, sql.builder.FromStruct(user))
			}

			err := sql.BatchCreate(ctx, "test_batch_users", records)
			So(err, ShouldBeNil)
		})

		Convey("测试 BatchCreate 的 CreateOption", func() {
			// 先创建一些记录
			users := []TestUser{
				{ID: 50, Name: "OriginalUser50", Age: 20, CreateAt: time.Now()},
				{ID: 51, Name: "OriginalUser51", Age: 21, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, sql.builder.FromStruct(user))
			}
			err := sql.BatchCreate(ctx, "test_batch_users", records)
			So(err, ShouldBeNil)

			// 测试批量创建时使用 IgnoreConflict 选项
			conflictUsers := []TestUser{
				{ID: 50, Name: "ConflictUser50", Age: 30, CreateAt: time.Now()}, // 冲突记录
				{ID: 52, Name: "NewUser52", Age: 22, CreateAt: time.Now()},      // 新记录
			}
			var conflictRecords []Record
			for _, user := range conflictUsers {
				conflictRecords = append(conflictRecords, sql.builder.FromStruct(user))
			}

			err = sql.BatchCreate(ctx, "test_batch_users", conflictRecords, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证ID=50的记录没有被修改，ID=52的记录被创建
			pk50 := map[string]any{"id": 50}
			result50, err := sql.Get(ctx, "test_batch_users", pk50)
			So(err, ShouldBeNil)
			var user50 TestUser
			result50.Scan(&user50)
			So(user50.Name, ShouldEqual, "OriginalUser50") // 原始记录没有被修改

			pk52 := map[string]any{"id": 52}
			result52, err := sql.Get(ctx, "test_batch_users", pk52)
			So(err, ShouldBeNil)
			var user52 TestUser
			result52.Scan(&user52)
			So(user52.Name, ShouldEqual, "NewUser52") // 新记录被创建
		})

		Convey("测试 BatchUpdate", func() {
			// 先创建记录
			users := []TestUser{
				{ID: 4, Name: "User4", Age: 23, CreateAt: time.Now()},
				{ID: 5, Name: "User5", Age: 24, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, sql.builder.FromStruct(user))
			}
			sql.BatchCreate(ctx, "test_batch_users", records)

			// 批量更新
			updatedUsers := []TestUser{
				{ID: 4, Name: "Updated User4", Age: 33, CreateAt: time.Now()},
				{ID: 5, Name: "Updated User5", Age: 34, CreateAt: time.Now()},
			}
			var updatedRecords []Record
			var pks []map[string]any
			for _, user := range updatedUsers {
				updatedRecords = append(updatedRecords, sql.builder.FromStruct(user))
				pks = append(pks, map[string]any{"id": user.ID})
			}

			err := sql.BatchUpdate(ctx, "test_batch_users", pks, updatedRecords)
			So(err, ShouldBeNil)
		})

		Convey("测试 BatchDelete", func() {
			// 先创建记录
			users := []TestUser{
				{ID: 6, Name: "User6", Age: 25, CreateAt: time.Now()},
				{ID: 7, Name: "User7", Age: 26, CreateAt: time.Now()},
			}
			var records []Record
			for _, user := range users {
				records = append(records, sql.builder.FromStruct(user))
			}
			sql.BatchCreate(ctx, "test_batch_users", records)

			// 批量删除
			pks := []map[string]any{
				{"id": 6},
				{"id": 7},
			}

			err := sql.BatchDelete(ctx, "test_batch_users", pks)
			So(err, ShouldBeNil)
		})
	})
}

func TestSQLTransaction(t *testing.T) {
	Convey("测试 SQL 事务操作", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		// 创建测试表
		ctx := context.Background()
		model := &TableModel{
			Table: "test_tx_users",
			Fields: []FieldDefinition{
				{Name: "id", Type: FieldTypeInt, Required: true},
				{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				{Name: "age", Type: FieldTypeInt},
				{Name: "active", Type: FieldTypeBool, Default: true},
				{Name: "score", Type: FieldTypeFloat},
				{Name: "email", Type: FieldTypeString, Size: 255},
				{Name: "create_at", Type: FieldTypeDate},
			},
			PrimaryKey: []string{"id"},
		}
		sql.Migrate(ctx, model)
		defer sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_tx_users")

		Convey("测试 BeginTx 和手动提交", func() {
			tx, err := sql.BeginTx(ctx)
			So(err, ShouldBeNil)
			So(tx, ShouldNotBeNil)

			user := TestUser{ID: 1, Name: "TxUser1", Age: 30, CreateAt: time.Now()}
			record := sql.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			err = tx.Commit()
			So(err, ShouldBeNil)

			// 验证提交成功
			pk := map[string]any{"id": 1}
			result, err := sql.Get(ctx, "test_tx_users", pk)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("测试事务回滚", func() {
			tx, err := sql.BeginTx(ctx)
			So(err, ShouldBeNil)

			user := TestUser{ID: 2, Name: "TxUser2", Age: 25, CreateAt: time.Now()}
			record := sql.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			err = tx.Rollback()
			So(err, ShouldBeNil)

			// 验证回滚成功
			pk := map[string]any{"id": 2}
			_, err = sql.Get(ctx, "test_tx_users", pk)
			So(err, ShouldEqual, ErrRecordNotFound)
		})

		Convey("测试 WithTx", func() {
			err := sql.WithTx(ctx, func(tx Transaction) error {
				user := TestUser{ID: 3, Name: "TxUser3", Age: 28, CreateAt: time.Now()}
				record := sql.builder.FromStruct(user)
				return tx.Create(ctx, "test_tx_users", record)
			})
			So(err, ShouldBeNil)

			// 验证提交成功
			pk := map[string]any{"id": 3}
			result, err := sql.Get(ctx, "test_tx_users", pk)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("测试事务中的 CreateOption", func() {
			tx, err := sql.BeginTx(ctx)
			So(err, ShouldBeNil)
			defer tx.Rollback()

			// 先创建一条记录
			user := TestUser{
				ID:       100,
				Name:     "TxOriginal",
				Email:    "txoriginal@example.com",
				Age:      25,
				Active:   true,
				Score:    85.0,
				CreateAt: time.Now(),
			}
			record := sql.builder.FromStruct(user)
			err = tx.Create(ctx, "test_tx_users", record)
			So(err, ShouldBeNil)

			// 测试 IgnoreConflict 选项
			conflictUser := TestUser{
				ID:       100,
				Name:     "TxConflict",
				Email:    "txconflict@example.com",
				Age:      30,
				Active:   false,
				Score:    95.0,
				CreateAt: time.Now(),
			}
			conflictRecord := sql.builder.FromStruct(conflictUser)
			
			err = tx.Create(ctx, "test_tx_users", conflictRecord, WithIgnoreConflict())
			So(err, ShouldBeNil)

			// 验证原始记录没有被修改（在事务中）
			pk := map[string]any{"id": 100}
			result, err := tx.Get(ctx, "test_tx_users", pk)
			So(err, ShouldBeNil)
			var retrievedUser TestUser
			result.Scan(&retrievedUser)
			So(retrievedUser.Name, ShouldEqual, "TxOriginal")
			So(retrievedUser.Email, ShouldEqual, "txoriginal@example.com")

			// 测试 UpdateOnConflict 选项
			updateUser := TestUser{
				ID:       100,
				Name:     "TxUpdated",
				Email:    "txupdated@example.com",
				Age:      35,
				Active:   false,
				Score:    99.0,
				CreateAt: time.Now(),
			}
			updateRecord := sql.builder.FromStruct(updateUser)
			
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

func TestSQLGetBuilder(t *testing.T) {
	Convey("测试 SQL GetBuilder 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		builder := sql.GetBuilder()
		So(builder, ShouldNotBeNil)
		So(builder, ShouldEqual, sql.builder)
	})
}

func TestSQLClose(t *testing.T) {
	Convey("测试 SQL Close 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)

		err = sql.Close()
		So(err, ShouldBeNil)
	})
}

func TestSQLBuildMethods(t *testing.T) {
	Convey("测试 SQL 构建方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		Convey("测试 buildCreateTableSQL", func() {
			model := &TableModel{
				Table: "test_build_table",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
					{Name: "email", Type: FieldTypeString, Size: 255},
					{Name: "age", Type: FieldTypeInt, Default: 0},
					{Name: "active", Type: FieldTypeBool, Default: true},
					{Name: "score", Type: FieldTypeFloat},
					{Name: "data", Type: FieldTypeJSON},
					{Name: "created_at", Type: FieldTypeDate},
				},
				PrimaryKey: []string{"id"},
			}

			sqlStr := sql.buildCreateTableSQL(model)
			So(sqlStr, ShouldContainSubstring, "CREATE TABLE IF NOT EXISTS test_build_table")
			So(sqlStr, ShouldContainSubstring, "id INT NOT NULL")
			So(sqlStr, ShouldContainSubstring, "name VARCHAR(100) NOT NULL")
			So(sqlStr, ShouldContainSubstring, "email VARCHAR(255)")
			So(sqlStr, ShouldContainSubstring, "age INT DEFAULT 0")
			So(sqlStr, ShouldContainSubstring, "active BOOLEAN DEFAULT 1")
			So(sqlStr, ShouldContainSubstring, "score FLOAT")
			So(sqlStr, ShouldContainSubstring, "data JSON")
			So(sqlStr, ShouldContainSubstring, "created_at DATETIME")
			So(sqlStr, ShouldContainSubstring, "PRIMARY KEY (id)")
		})

		Convey("测试 buildColumnDefinition", func() {
			field := FieldDefinition{
				Name:     "test_field",
				Type:     FieldTypeString,
				Size:     50,
				Required: true,
				Default:  "default_value",
			}

			columnDef := sql.buildColumnDefinition(field)
			So(columnDef, ShouldEqual, "test_field VARCHAR(50) NOT NULL DEFAULT 'default_value'")
		})

		Convey("测试 mapFieldTypeToSQL", func() {
			So(sql.mapFieldTypeToSQL(FieldTypeString, 100), ShouldEqual, "VARCHAR(100)")
			So(sql.mapFieldTypeToSQL(FieldTypeString, 0), ShouldEqual, "VARCHAR(255)")
			So(sql.mapFieldTypeToSQL(FieldTypeInt, 0), ShouldEqual, "INT")
			So(sql.mapFieldTypeToSQL(FieldTypeFloat, 0), ShouldEqual, "FLOAT")
			So(sql.mapFieldTypeToSQL(FieldTypeBool, 0), ShouldEqual, "BOOLEAN")
			So(sql.mapFieldTypeToSQL(FieldTypeDate, 0), ShouldEqual, "DATETIME")
			So(sql.mapFieldTypeToSQL(FieldTypeJSON, 0), ShouldEqual, "JSON")
		})

		Convey("测试 formatDefaultValue", func() {
			So(sql.formatDefaultValue("test"), ShouldEqual, "'test'")
			So(sql.formatDefaultValue("test's"), ShouldEqual, "'test''s'")
			So(sql.formatDefaultValue(true), ShouldEqual, "1")
			So(sql.formatDefaultValue(false), ShouldEqual, "0")
			So(sql.formatDefaultValue(123), ShouldEqual, "123")
			So(sql.formatDefaultValue(12.34), ShouldEqual, "12.34")
		})

		Convey("测试 buildCreateIndexSQL", func() {
			index := IndexDefinition{
				Name:   "idx_test",
				Fields: []string{"name", "age"},
				Unique: false,
			}
			indexSQL := sql.buildCreateIndexSQL("test_table", index)
			So(indexSQL, ShouldEqual, "CREATE INDEX idx_test ON test_table (name, age)")

			uniqueIndex := IndexDefinition{
				Name:   "idx_unique_email",
				Fields: []string{"email"},
				Unique: true,
			}
			uniqueIndexSQL := sql.buildCreateIndexSQL("test_table", uniqueIndex)
			So(uniqueIndexSQL, ShouldEqual, "CREATE UNIQUE INDEX idx_unique_email ON test_table (email)")
		})
	})
}

func TestSQLFormatSQL(t *testing.T) {
	Convey("测试 formatSQL 方法", t, func() {
		Convey("MySQL 驱动", func() {
			sql, err := NewSQLWithOptions(testMySQLOptions)
			So(err, ShouldBeNil)
			defer sql.Close()

			sqlStr := "SELECT * FROM users WHERE id = ? AND name = ?"
			args := []any{1, "John"}

			formattedSQL, formattedArgs := sql.formatSQL(sqlStr, args)
			So(formattedSQL, ShouldEqual, "SELECT * FROM users WHERE id = ? AND name = ?")
			So(formattedArgs, ShouldResemble, []any{1, "John"})
		})

		Convey("PostgreSQL 驱动 (模拟)", func() {
			// 创建一个模拟的 PostgreSQL SQL 实例
			sql := &SQL{driver: "postgres"}

			sqlStr := "SELECT * FROM users WHERE id = ? AND name = ?"
			args := []any{1, "John"}

			formattedSQL, formattedArgs := sql.formatSQL(sqlStr, args)
			So(formattedSQL, ShouldEqual, "SELECT * FROM users WHERE id = $1 AND name = $2")
			So(formattedArgs, ShouldResemble, []any{1, "John"})
		})
	})
}

func TestSQLTransactionMethods(t *testing.T) {
	Convey("测试 SQLTransaction 特有方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		ctx := context.Background()
		tx, err := sql.BeginTx(ctx)
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
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100},
				},
				PrimaryKey: []string{"id"},
			}

			err := tx.Migrate(ctx, model)
			So(err, ShouldBeNil)
		})
	})
}

func TestSQLErrorHandling(t *testing.T) {
	Convey("测试 SQL 错误处理", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		ctx := context.Background()

		Convey("测试获取不存在的记录", func() {
			pk := map[string]any{"id": 99999}
			_, err := sql.Get(ctx, "non_existent_table", pk)
			So(err, ShouldNotBeNil)
		})

		Convey("测试在不存在的表上创建记录", func() {
			user := TestUser{ID: 1, Name: "Test"}
			record := sql.builder.FromStruct(user)
			err := sql.Create(ctx, "non_existent_table", record)
			So(err, ShouldNotBeNil)
		})

		Convey("测试 mapToStruct 错误情况", func() {
			data := map[string]any{"id": "not_a_number"}
			var user TestUser
			// 这个测试可能会成功，因为 Go 的反射会尝试类型转换
			// 但我们至少验证了函数不会 panic
			So(func() { mapToStruct(data, &user) }, ShouldNotPanic)
		})
	})
}

func TestSQLEdgeCases(t *testing.T) {
	Convey("测试 SQL 边界情况", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		Convey("测试空字段的结构体", func() {
			type EmptyStruct struct{}
			empty := EmptyStruct{}
			result := structToMap(empty)
			So(len(result), ShouldEqual, 0)
		})

		Convey("测试带有未导出字段的结构体", func() {
			type StructWithPrivateFields struct {
				ID           int    `rdb:"id"`
				privateField string // 未导出字段
				Name         string `rdb:"name"`
			}

			s := StructWithPrivateFields{
				ID:           1,
				privateField: "private",
				Name:         "test",
			}

			result := structToMap(s)
			So(result["id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "test")
			So(result["privateField"], ShouldBeNil) // 未导出字段不应该被包含
		})

		Convey("测试带有 rdb:'-' 标签的字段", func() {
			type StructWithIgnoredField struct {
				ID      int    `rdb:"id"`
				Ignored string `rdb:"-"`
				Name    string `rdb:"name"`
			}

			s := StructWithIgnoredField{
				ID:      1,
				Ignored: "ignored",
				Name:    "test",
			}

			result := structToMap(s)
			So(result["id"], ShouldEqual, 1)
			So(result["name"], ShouldEqual, "test")
			_, exists := result["Ignored"]
			So(exists, ShouldBeFalse) // 被忽略的字段不应该被包含
		})

		Convey("测试复合主键", func() {
			model := &TableModel{
				Table: "test_composite_pk",
				Fields: []FieldDefinition{
					{Name: "user_id", Type: FieldTypeInt, Required: true},
					{Name: "role_id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100},
				},
				PrimaryKey: []string{"user_id", "role_id"},
			}

			ctx := context.Background()
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 清理
			sql.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_composite_pk")
		})

		Convey("测试批量操作长度不匹配", func() {
			pks := []map[string]any{{"id": 1}, {"id": 2}}
			records := []Record{sql.builder.FromStruct(TestUser{ID: 1})} // 只有一个记录

			ctx := context.Background()
			err := sql.BatchUpdate(ctx, "test_table", pks, records)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "length mismatch")
		})
	})
}

func TestSQLDropTable(t *testing.T) {
	Convey("测试 SQL DropTable 方法", t, func() {
		sql, err := NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer sql.Close()

		ctx := context.Background()

		Convey("删除存在的表", func() {
			// 先创建一个测试表
			model := &TableModel{
				Table: "test_drop_table_exists",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				},
				PrimaryKey: []string{"id"},
			}
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 删除表
			err = sql.DropTable(ctx, "test_drop_table_exists")
			So(err, ShouldBeNil)

			// 验证表已被删除 - 尝试在已删除的表上执行操作应该失败
			user := TestUser{ID: 1, Name: "Test"}
			record := sql.builder.FromStruct(user)
			err = sql.Create(ctx, "test_drop_table_exists", record)
			So(err, ShouldNotBeNil)
		})

		Convey("删除不存在的表", func() {
			// 删除不存在的表应该不会报错（使用 IF EXISTS）
			err := sql.DropTable(ctx, "test_drop_table_not_exists")
			So(err, ShouldBeNil)
		})

		Convey("在事务中删除表", func() {
			// 先创建一个测试表
			model := &TableModel{
				Table: "test_drop_table_tx",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				},
				PrimaryKey: []string{"id"},
			}
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 在事务中删除表
			tx, err := sql.BeginTx(ctx)
			So(err, ShouldBeNil)

			err = tx.DropTable(ctx, "test_drop_table_tx")
			So(err, ShouldBeNil)

			err = tx.Commit()
			So(err, ShouldBeNil)

			// 验证表已被删除
			user := TestUser{ID: 1, Name: "Test"}
			record := sql.builder.FromStruct(user)
			err = sql.Create(ctx, "test_drop_table_tx", record)
			So(err, ShouldNotBeNil)
		})

		Convey("在事务中删除表后回滚", func() {
			// 注意：在 MySQL 中，DDL 操作（如 DROP TABLE）会自动提交事务，无法回滚
			// 这里测试验证 DDL 操作的自动提交行为
			
			// 先创建一个测试表
			model := &TableModel{
				Table: "test_drop_table_rollback",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				},
				PrimaryKey: []string{"id"},
			}
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 在事务中删除表
			tx, err := sql.BeginTx(ctx)
			So(err, ShouldBeNil)

			err = tx.DropTable(ctx, "test_drop_table_rollback")
			So(err, ShouldBeNil)

			// 回滚事务（但 DDL 操作已经自动提交了）
			err = tx.Rollback()
			So(err, ShouldBeNil)

			// 验证表已被删除（DDL 操作无法回滚）
			user := TestUser{ID: 1, Name: "Test"}
			record := sql.builder.FromStruct(user)
			err = sql.Create(ctx, "test_drop_table_rollback", record)
			So(err, ShouldNotBeNil) // 表应该已被删除
		})

		Convey("使用 WithTx 删除表", func() {
			// 先创建一个测试表
			model := &TableModel{
				Table: "test_drop_table_with_tx",
				Fields: []FieldDefinition{
					{Name: "id", Type: FieldTypeInt, Required: true},
					{Name: "name", Type: FieldTypeString, Size: 100, Required: true},
				},
				PrimaryKey: []string{"id"},
			}
			err := sql.Migrate(ctx, model)
			So(err, ShouldBeNil)

			// 使用 WithTx 删除表
			err = sql.WithTx(ctx, func(tx Transaction) error {
				return tx.DropTable(ctx, "test_drop_table_with_tx")
			})
			So(err, ShouldBeNil)

			// 验证表已被删除
			user := TestUser{ID: 1, Name: "Test"}
			record := sql.builder.FromStruct(user)
			err = sql.Create(ctx, "test_drop_table_with_tx", record)
			So(err, ShouldNotBeNil)
		})
	})
}
