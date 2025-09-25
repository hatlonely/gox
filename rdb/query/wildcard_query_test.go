package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWildcardQueryType(t *testing.T) {
	Convey("测试 WildcardQuery Type 方法", t, func() {
		q := &WildcardQuery{Field: "name", Value: "test*"}
		So(q.Type(), ShouldEqual, QueryTypeWildcard)
	})
}

func TestWildcardQueryToES(t *testing.T) {
	Convey("测试 WildcardQuery ToES 方法", t, func() {
		Convey("通配符 *", func() {
			q := &WildcardQuery{
				Field: "name",
				Value: "test*",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"wildcard": map[string]interface{}{
					"name": "test*",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("通配符 ?", func() {
			q := &WildcardQuery{
				Field: "code",
				Value: "AB?D",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"wildcard": map[string]interface{}{
					"code": "AB?D",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("混合通配符", func() {
			q := &WildcardQuery{
				Field: "pattern",
				Value: "test*?end",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"wildcard": map[string]interface{}{
					"pattern": "test*?end",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("无通配符", func() {
			q := &WildcardQuery{
				Field: "exact",
				Value: "testvalue",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"wildcard": map[string]interface{}{
					"exact": "testvalue",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestWildcardQueryToSQL(t *testing.T) {
	Convey("测试 WildcardQuery ToSQL 方法", t, func() {
		Convey("通配符 * 转换为 %", func() {
			q := &WildcardQuery{
				Field: "name",
				Value: "test*",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "name LIKE ?")
			So(args, ShouldResemble, []interface{}{"test%"})
		})

		Convey("通配符 ? 转换为 _", func() {
			q := &WildcardQuery{
				Field: "code",
				Value: "AB?D",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "code LIKE ?")
			So(args, ShouldResemble, []interface{}{"AB_D"})
		})

		Convey("混合通配符", func() {
			q := &WildcardQuery{
				Field: "pattern",
				Value: "test*?end",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "pattern LIKE ?")
			So(args, ShouldResemble, []interface{}{"test%_end"})
		})

		Convey("无通配符", func() {
			q := &WildcardQuery{
				Field: "exact",
				Value: "testvalue",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "exact LIKE ?")
			So(args, ShouldResemble, []interface{}{"testvalue"})
		})

		Convey("空值", func() {
			q := &WildcardQuery{
				Field: "empty",
				Value: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "empty LIKE ?")
			So(args, ShouldResemble, []interface{}{""})
		})
	})
}

func TestWildcardQueryToMongo(t *testing.T) {
	Convey("测试 WildcardQuery ToMongo 方法", t, func() {
		Convey("通配符 * 转换为 .*", func() {
			q := &WildcardQuery{
				Field: "name",
				Value: "test*",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex":   "^test.*$",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("通配符 ? 转换为 .", func() {
			q := &WildcardQuery{
				Field: "code",
				Value: "AB?D",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"code": map[string]interface{}{
					"$regex":   "^AB.D$",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("混合通配符", func() {
			q := &WildcardQuery{
				Field: "pattern",
				Value: "test*?end",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"pattern": map[string]interface{}{
					"$regex":   "^test.*.end$",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("无通配符", func() {
			q := &WildcardQuery{
				Field: "exact",
				Value: "testvalue",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"exact": map[string]interface{}{
					"$regex":   "^testvalue$",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空值", func() {
			q := &WildcardQuery{
				Field: "empty",
				Value: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"empty": map[string]interface{}{
					"$regex":   "^$",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}