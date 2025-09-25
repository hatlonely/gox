package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPrefixQueryType(t *testing.T) {
	Convey("测试 PrefixQuery Type 方法", t, func() {
		q := &PrefixQuery{Field: "name", Value: "test"}
		So(q.Type(), ShouldEqual, QueryTypePrefix)
	})
}

func TestPrefixQueryToES(t *testing.T) {
	Convey("测试 PrefixQuery ToES 方法", t, func() {
		Convey("普通前缀", func() {
			q := &PrefixQuery{
				Field: "name",
				Value: "test",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"prefix": map[string]interface{}{
					"name": "test",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("单字符前缀", func() {
			q := &PrefixQuery{
				Field: "code",
				Value: "A",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"prefix": map[string]interface{}{
					"code": "A",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字前缀", func() {
			q := &PrefixQuery{
				Field: "id",
				Value: "123",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"prefix": map[string]interface{}{
					"id": "123",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空前缀", func() {
			q := &PrefixQuery{
				Field: "empty",
				Value: "",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"prefix": map[string]interface{}{
					"empty": "",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestPrefixQueryToSQL(t *testing.T) {
	Convey("测试 PrefixQuery ToSQL 方法", t, func() {
		Convey("普通前缀", func() {
			q := &PrefixQuery{
				Field: "name",
				Value: "test",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "name LIKE ?")
			So(args, ShouldResemble, []interface{}{"test%"})
		})

		Convey("单字符前缀", func() {
			q := &PrefixQuery{
				Field: "code",
				Value: "A",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "code LIKE ?")
			So(args, ShouldResemble, []interface{}{"A%"})
		})

		Convey("数字前缀", func() {
			q := &PrefixQuery{
				Field: "id",
				Value: "123",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "id LIKE ?")
			So(args, ShouldResemble, []interface{}{"123%"})
		})

		Convey("空前缀", func() {
			q := &PrefixQuery{
				Field: "empty",
				Value: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "empty LIKE ?")
			So(args, ShouldResemble, []interface{}{"%"})
		})

		Convey("包含特殊字符的前缀", func() {
			q := &PrefixQuery{
				Field: "path",
				Value: "/usr/local",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "path LIKE ?")
			So(args, ShouldResemble, []interface{}{"/usr/local%"})
		})
	})
}

func TestPrefixQueryToMongo(t *testing.T) {
	Convey("测试 PrefixQuery ToMongo 方法", t, func() {
		Convey("普通前缀", func() {
			q := &PrefixQuery{
				Field: "name",
				Value: "test",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex":   "^test",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("单字符前缀", func() {
			q := &PrefixQuery{
				Field: "code",
				Value: "A",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"code": map[string]interface{}{
					"$regex":   "^A",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字前缀", func() {
			q := &PrefixQuery{
				Field: "id",
				Value: "123",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"id": map[string]interface{}{
					"$regex":   "^123",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空前缀", func() {
			q := &PrefixQuery{
				Field: "empty",
				Value: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"empty": map[string]interface{}{
					"$regex":   "^",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("包含特殊字符的前缀", func() {
			q := &PrefixQuery{
				Field: "path",
				Value: "/usr/local",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"path": map[string]interface{}{
					"$regex":   "^/usr/local",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}