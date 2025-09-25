package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExistsQueryType(t *testing.T) {
	Convey("测试 ExistsQuery Type 方法", t, func() {
		q := &ExistsQuery{Field: "email"}
		So(q.Type(), ShouldEqual, QueryTypeExists)
	})
}

func TestExistsQueryToES(t *testing.T) {
	Convey("测试 ExistsQuery ToES 方法", t, func() {
		Convey("基本字段存在查询", func() {
			q := &ExistsQuery{
				Field: "email",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"exists": map[string]interface{}{
					"field": "email",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("嵌套字段存在查询", func() {
			q := &ExistsQuery{
				Field: "user.profile.email",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"exists": map[string]interface{}{
					"field": "user.profile.email",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数组字段存在查询", func() {
			q := &ExistsQuery{
				Field: "tags",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"exists": map[string]interface{}{
					"field": "tags",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字段名", func() {
			q := &ExistsQuery{
				Field: "",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"exists": map[string]interface{}{
					"field": "",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestExistsQueryToSQL(t *testing.T) {
	Convey("测试 ExistsQuery ToSQL 方法", t, func() {
		Convey("基本字段存在查询", func() {
			q := &ExistsQuery{
				Field: "email",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "email IS NOT NULL")
			So(args, ShouldBeEmpty)
		})

		Convey("带点号的字段名", func() {
			q := &ExistsQuery{
				Field: "user.email",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "user.email IS NOT NULL")
			So(args, ShouldBeEmpty)
		})

		Convey("带下划线的字段名", func() {
			q := &ExistsQuery{
				Field: "created_at",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "created_at IS NOT NULL")
			So(args, ShouldBeEmpty)
		})

		Convey("数字开头的字段名", func() {
			q := &ExistsQuery{
				Field: "2fa_enabled",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "2fa_enabled IS NOT NULL")
			So(args, ShouldBeEmpty)
		})

		Convey("空字段名", func() {
			q := &ExistsQuery{
				Field: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, " IS NOT NULL")
			So(args, ShouldBeEmpty)
		})
	})
}

func TestExistsQueryToMongo(t *testing.T) {
	Convey("测试 ExistsQuery ToMongo 方法", t, func() {
		Convey("基本字段存在查询", func() {
			q := &ExistsQuery{
				Field: "email",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"email": map[string]interface{}{
					"$exists": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("嵌套字段存在查询", func() {
			q := &ExistsQuery{
				Field: "user.profile.email",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"user.profile.email": map[string]interface{}{
					"$exists": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数组字段存在查询", func() {
			q := &ExistsQuery{
				Field: "tags",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"tags": map[string]interface{}{
					"$exists": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("带特殊字符的字段名", func() {
			q := &ExistsQuery{
				Field: "user-profile.email_address",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"user-profile.email_address": map[string]interface{}{
					"$exists": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字段名", func() {
			q := &ExistsQuery{
				Field: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"": map[string]interface{}{
					"$exists": true,
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}