package rdb

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestSQLSimple(t *testing.T) {
	convey.Convey("简单 SQL 测试", t, func() {
		// 每次都创建新的数据库连接
		options := &SQLOptions{
			Driver:   "sqlite3",
			Database: ":memory:",
		}

		sql, err := NewSQLWithOptions(options)
		convey.So(err, convey.ShouldBeNil)
		defer sql.Close()

		ctx := context.Background()

		// 创建表
		_, err = sql.db.ExecContext(ctx, `
			CREATE TABLE test_users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				email TEXT NOT NULL,
				age INTEGER
			)
		`)
		convey.So(err, convey.ShouldBeNil)

		// 测试 Create
		builder := sql.GetBuilder()
		user := map[string]any{
			"name":  "测试用户",
			"email": "test@example.com",
			"age":   30,
		}
		record := builder.FromMap(user, "test_users")
		
		err = sql.Create(ctx, "test_users", record)
		convey.So(err, convey.ShouldBeNil)

		// 测试 Get
		pk := map[string]any{"id": 1}
		retrievedRecord, err := sql.Get(ctx, "test_users", pk)
		convey.So(err, convey.ShouldBeNil)
		convey.So(retrievedRecord, convey.ShouldNotBeNil)

		fields := retrievedRecord.Fields()
		convey.So(fields["name"], convey.ShouldEqual, "测试用户")
		convey.So(fields["email"], convey.ShouldEqual, "test@example.com")
	})
}