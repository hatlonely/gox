package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTermQueryType(t *testing.T) {
	Convey("测试 TermQuery Type 方法", t, func() {
		q := &TermQuery{Field: "status", Value: "active"}
		So(q.Type(), ShouldEqual, QueryTypeTerm)
	})
}

func TestTermQueryToES(t *testing.T) {
	Convey("测试 TermQuery ToES 方法", t, func() {
		Convey("字符串值", func() {
			q := &TermQuery{
				Field: "status",
				Value: "active",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"status": "active",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字值", func() {
			q := &TermQuery{
				Field: "age",
				Value: 25,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"age": 25,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("布尔值", func() {
			q := &TermQuery{
				Field: "active",
				Value: true,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"active": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("浮点数值", func() {
			q := &TermQuery{
				Field: "score",
				Value: 95.5,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"score": 95.5,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字符串值", func() {
			q := &TermQuery{
				Field: "description",
				Value: "",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"description": "",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("nil 值", func() {
			q := &TermQuery{
				Field: "optional_field",
				Value: nil,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"term": map[string]interface{}{
					"optional_field": nil,
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestTermQueryToSQL(t *testing.T) {
	Convey("测试 TermQuery ToSQL 方法", t, func() {
		Convey("字符串值", func() {
			q := &TermQuery{
				Field: "status",
				Value: "active",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "status = ?")
			So(args, ShouldResemble, []interface{}{"active"})
		})

		Convey("数字值", func() {
			q := &TermQuery{
				Field: "age",
				Value: 25,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age = ?")
			So(args, ShouldResemble, []interface{}{25})
		})

		Convey("布尔值", func() {
			q := &TermQuery{
				Field: "active",
				Value: true,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "active = ?")
			So(args, ShouldResemble, []interface{}{true})
		})

		Convey("浮点数值", func() {
			q := &TermQuery{
				Field: "score",
				Value: 95.5,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "score = ?")
			So(args, ShouldResemble, []interface{}{95.5})
		})

		Convey("空字符串值", func() {
			q := &TermQuery{
				Field: "description",
				Value: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "description = ?")
			So(args, ShouldResemble, []interface{}{""})
		})

		Convey("nil 值", func() {
			q := &TermQuery{
				Field: "optional_field",
				Value: nil,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "optional_field = ?")
			So(args, ShouldResemble, []interface{}{nil})
		})
	})
}

func TestTermQueryToMongo(t *testing.T) {
	Convey("测试 TermQuery ToMongo 方法", t, func() {
		Convey("字符串值", func() {
			q := &TermQuery{
				Field: "status",
				Value: "active",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"status": "active",
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字值", func() {
			q := &TermQuery{
				Field: "age",
				Value: 25,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": 25,
			}
			So(result, ShouldResemble, expected)
		})

		Convey("布尔值", func() {
			q := &TermQuery{
				Field: "active",
				Value: true,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"active": true,
			}
			So(result, ShouldResemble, expected)
		})

		Convey("浮点数值", func() {
			q := &TermQuery{
				Field: "score",
				Value: 95.5,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"score": 95.5,
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字符串值", func() {
			q := &TermQuery{
				Field: "description",
				Value: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"description": "",
			}
			So(result, ShouldResemble, expected)
		})

		Convey("nil 值", func() {
			q := &TermQuery{
				Field: "optional_field",
				Value: nil,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"optional_field": nil,
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数组值", func() {
			q := &TermQuery{
				Field: "tags",
				Value: []string{"tag1", "tag2"},
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"tags": []string{"tag1", "tag2"},
			}
			So(result, ShouldResemble, expected)
		})
	})
}