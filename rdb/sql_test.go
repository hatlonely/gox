package rdb

import (
	"context"
	"testing"

	"github.com/hatlonely/gox/rdb/query"
	"github.com/smartystreets/goconvey/convey"
)

func TestSQL_Basic(t *testing.T) {
	convey.Convey("测试 SQL 基础功能", t, func() {
		// 使用 SQLite 内存数据库进行测试
		options := &SQLOptions{
			Driver:   "sqlite3",
			Database: ":memory:",
		}

		sql, err := NewSQLWithOptions(options)
		convey.So(err, convey.ShouldBeNil)
		convey.So(sql, convey.ShouldNotBeNil)
		defer sql.Close()

		ctx := context.Background()

		// 辅助函数：创建测试表
		createTable := func() {
			_, err := sql.db.ExecContext(ctx, `DROP TABLE IF EXISTS users`)
			convey.So(err, convey.ShouldBeNil)
			_, err = sql.db.ExecContext(ctx, `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					email TEXT NOT NULL,
					age INTEGER
				)
			`)
			convey.So(err, convey.ShouldBeNil)
		}

		convey.Convey("测试 RecordBuilder", func() {
			builder := sql.GetBuilder()
			convey.So(builder, convey.ShouldNotBeNil)

			// 测试 FromStruct
			user := struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Age   int    `json:"age"`
			}{
				Name:  "张三",
				Email: "zhang@example.com",
				Age:   25,
			}

			record := builder.FromStruct(user)
			convey.So(record, convey.ShouldNotBeNil)

			fields := record.Fields()
			convey.So(fields["name"], convey.ShouldEqual, "张三")
			convey.So(fields["email"], convey.ShouldEqual, "zhang@example.com")
			convey.So(fields["age"], convey.ShouldEqual, 25)

			// 测试 FromMap
			data := map[string]any{
				"name":  "李四",
				"email": "li@example.com",
				"age":   30,
			}
			record2 := builder.FromMap(data, "users")
			convey.So(record2, convey.ShouldNotBeNil)
			convey.So(record2.Fields(), convey.ShouldResemble, data)
		})

		convey.Convey("测试 CRUD 操作", func() {
			createTable()
			builder := sql.GetBuilder()

			// 测试 Create
			user := map[string]any{
				"name":  "张三",
				"email": "zhang@example.com",
				"age":   25,
			}
			record := builder.FromMap(user, "users")
			err := sql.Create(ctx, "users", record)
			convey.So(err, convey.ShouldBeNil)

			// 测试 Get
			pk := map[string]any{"id": 1}
			retrievedRecord, err := sql.Get(ctx, "users", pk)
			convey.So(err, convey.ShouldBeNil)
			convey.So(retrievedRecord, convey.ShouldNotBeNil)

			fields := retrievedRecord.Fields()
			convey.So(fields["name"], convey.ShouldEqual, "张三")
			convey.So(fields["email"], convey.ShouldEqual, "zhang@example.com")

			// 测试 Update
			updatedUser := map[string]any{
				"name":  "张三丰",
				"email": "zhangsanfeng@example.com",
				"age":   26,
			}
			updateRecord := builder.FromMap(updatedUser, "users")
			err = sql.Update(ctx, "users", pk, updateRecord)
			convey.So(err, convey.ShouldBeNil)

			// 验证更新
			retrievedRecord, err = sql.Get(ctx, "users", pk)
			convey.So(err, convey.ShouldBeNil)
			fields = retrievedRecord.Fields()
			convey.So(fields["name"], convey.ShouldEqual, "张三丰")

			// 测试 Delete
			err = sql.Delete(ctx, "users", pk)
			convey.So(err, convey.ShouldBeNil)

			// 验证删除
			_, err = sql.Get(ctx, "users", pk)
			convey.So(err, convey.ShouldEqual, ErrRecordNotFound)
		})

		convey.Convey("测试查询功能", func() {
			createTable()
			builder := sql.GetBuilder()

			// 创建测试数据
			users := []map[string]any{
				{"name": "张三", "email": "zhang@example.com", "age": 25},
				{"name": "李四", "email": "li@example.com", "age": 30},
				{"name": "王五", "email": "wang@example.com", "age": 35},
			}

			for _, user := range users {
				record := builder.FromMap(user, "users")
				err := sql.Create(ctx, "users", record)
				convey.So(err, convey.ShouldBeNil)
			}

			// 测试简单查询
			termQuery := &query.TermQuery{
				Field: "name",
				Value: "张三",
			}

			records, err := sql.Find(ctx, "users", termQuery)
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(records), convey.ShouldEqual, 1)
			convey.So(records[0].Fields()["name"], convey.ShouldEqual, "张三")

			// 测试范围查询
			rangeQuery := &query.RangeQuery{
				Field: "age",
				Gte:   25,
				Lte:   30,
			}

			records, err = sql.Find(ctx, "users", rangeQuery)
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(records), convey.ShouldBeGreaterThanOrEqualTo, 2)
		})

		convey.Convey("测试事务功能", func() {
			createTable()
			builder := sql.GetBuilder()

			err := sql.WithTx(ctx, func(tx Transaction) error {
				// 在事务中创建用户
				user := map[string]any{
					"name":  "事务用户",
					"email": "tx@example.com",
					"age":   40,
				}
				record := builder.FromMap(user, "users")
				return tx.Create(ctx, "users", record)
			})
			convey.So(err, convey.ShouldBeNil)

			// 验证事务提交后数据存在
			termQuery := &query.TermQuery{
				Field: "name",
				Value: "事务用户",
			}
			records, err := sql.Find(ctx, "users", termQuery)
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(records), convey.ShouldEqual, 1)
		})
	})
}

func TestSQLRecord_Scan(t *testing.T) {
	convey.Convey("测试 SQLRecord 扫描功能", t, func() {
		data := map[string]any{
			"id":    int64(1),
			"name":  "张三",
			"email": "zhang@example.com",
			"age":   int64(25),
		}

		record := &SQLRecord{data: data}

		convey.Convey("测试扫描到结构体", func() {
			type User struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
				Age   int64  `json:"age"`
			}

			var user User
			err := record.Scan(&user)
			convey.So(err, convey.ShouldBeNil)
			convey.So(user.ID, convey.ShouldEqual, 1)
			convey.So(user.Name, convey.ShouldEqual, "张三")
			convey.So(user.Email, convey.ShouldEqual, "zhang@example.com")
			convey.So(user.Age, convey.ShouldEqual, 25)
		})

		convey.Convey("测试获取字段", func() {
			fields := record.Fields()
			convey.So(fields, convey.ShouldResemble, data)
		})
	})
}