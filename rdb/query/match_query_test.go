package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMatchQueryType(t *testing.T) {
	Convey("测试 MatchQuery Type 方法", t, func() {
		q := &MatchQuery{Field: "title", Value: "search term"}
		So(q.Type(), ShouldEqual, QueryTypeMatch)
	})
}

func TestMatchQueryToES(t *testing.T) {
	Convey("测试 MatchQuery ToES 方法", t, func() {
		Convey("字符串值", func() {
			q := &MatchQuery{
				Field: "title",
				Value: "search term",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"match": map[string]interface{}{
					"title": "search term",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字值", func() {
			q := &MatchQuery{
				Field: "age",
				Value: 25,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"match": map[string]interface{}{
					"age": 25,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("布尔值", func() {
			q := &MatchQuery{
				Field: "active",
				Value: true,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"match": map[string]interface{}{
					"active": true,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字符串值", func() {
			q := &MatchQuery{
				Field: "description",
				Value: "",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"match": map[string]interface{}{
					"description": "",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestMatchQueryToSQL(t *testing.T) {
	Convey("测试 MatchQuery ToSQL 方法", t, func() {
		Convey("字符串值", func() {
			q := &MatchQuery{
				Field: "title",
				Value: "search term",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "title LIKE ?")
			So(args, ShouldResemble, []interface{}{"%search term%"})
		})

		Convey("数字值", func() {
			q := &MatchQuery{
				Field: "age",
				Value: 25,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age LIKE ?")
			So(args, ShouldResemble, []interface{}{"%25%"})
		})

		Convey("布尔值", func() {
			q := &MatchQuery{
				Field: "active",
				Value: true,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "active LIKE ?")
			So(args, ShouldResemble, []interface{}{"%true%"})
		})

		Convey("空字符串值", func() {
			q := &MatchQuery{
				Field: "description",
				Value: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "description LIKE ?")
			So(args, ShouldResemble, []interface{}{"%%"})
		})

		Convey("包含特殊字符的值", func() {
			q := &MatchQuery{
				Field: "content",
				Value: "hello world!",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "content LIKE ?")
			So(args, ShouldResemble, []interface{}{"%hello world!%"})
		})
	})
}

func TestMatchQueryToMongo(t *testing.T) {
	Convey("测试 MatchQuery ToMongo 方法", t, func() {
		Convey("字符串值", func() {
			q := &MatchQuery{
				Field: "title",
				Value: "search term",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"title": map[string]interface{}{
					"$regex":   "search term",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字值", func() {
			q := &MatchQuery{
				Field: "age",
				Value: 25,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{
					"$regex":   25,
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("布尔值", func() {
			q := &MatchQuery{
				Field: "active",
				Value: true,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"active": map[string]interface{}{
					"$regex":   true,
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空字符串值", func() {
			q := &MatchQuery{
				Field: "description",
				Value: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"description": map[string]interface{}{
					"$regex":   "",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("包含正则表达式特殊字符的值", func() {
			q := &MatchQuery{
				Field: "pattern",
				Value: "test.*pattern",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"pattern": map[string]interface{}{
					"$regex":   "test.*pattern",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}