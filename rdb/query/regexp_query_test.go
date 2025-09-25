package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRegexpQueryType(t *testing.T) {
	Convey("测试 RegexpQuery Type 方法", t, func() {
		q := &RegexpQuery{Field: "name", Value: "test.*"}
		So(q.Type(), ShouldEqual, QueryTypeRegexp)
	})
}

func TestRegexpQueryToES(t *testing.T) {
	Convey("测试 RegexpQuery ToES 方法", t, func() {
		Convey("简单正则表达式", func() {
			q := &RegexpQuery{
				Field: "name",
				Value: "test.*",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"regexp": map[string]interface{}{
					"name": "test.*",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字匹配正则", func() {
			q := &RegexpQuery{
				Field: "phone",
				Value: "\\d{11}",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"regexp": map[string]interface{}{
					"phone": "\\d{11}",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("邮箱匹配正则", func() {
			q := &RegexpQuery{
				Field: "email",
				Value: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"regexp": map[string]interface{}{
					"email": "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("字符类正则", func() {
			q := &RegexpQuery{
				Field: "code",
				Value: "[A-Z]{2}[0-9]{4}",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"regexp": map[string]interface{}{
					"code": "[A-Z]{2}[0-9]{4}",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空正则表达式", func() {
			q := &RegexpQuery{
				Field: "empty",
				Value: "",
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"regexp": map[string]interface{}{
					"empty": "",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}

func TestRegexpQueryToSQL(t *testing.T) {
	Convey("测试 RegexpQuery ToSQL 方法", t, func() {
		Convey("简单正则表达式", func() {
			q := &RegexpQuery{
				Field: "name",
				Value: "test.*",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "name REGEXP ?")
			So(args, ShouldResemble, []interface{}{"test.*"})
		})

		Convey("数字匹配正则", func() {
			q := &RegexpQuery{
				Field: "phone",
				Value: "\\d{11}",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "phone REGEXP ?")
			So(args, ShouldResemble, []interface{}{"\\d{11}"})
		})

		Convey("邮箱匹配正则", func() {
			q := &RegexpQuery{
				Field: "email",
				Value: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "email REGEXP ?")
			So(args, ShouldResemble, []interface{}{"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"})
		})

		Convey("字符类正则", func() {
			q := &RegexpQuery{
				Field: "code",
				Value: "[A-Z]{2}[0-9]{4}",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "code REGEXP ?")
			So(args, ShouldResemble, []interface{}{"[A-Z]{2}[0-9]{4}"})
		})

		Convey("空正则表达式", func() {
			q := &RegexpQuery{
				Field: "empty",
				Value: "",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "empty REGEXP ?")
			So(args, ShouldResemble, []interface{}{""})
		})

		Convey("OR 选择正则", func() {
			q := &RegexpQuery{
				Field: "status",
				Value: "active|pending|completed",
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "status REGEXP ?")
			So(args, ShouldResemble, []interface{}{"active|pending|completed"})
		})
	})
}

func TestRegexpQueryToMongo(t *testing.T) {
	Convey("测试 RegexpQuery ToMongo 方法", t, func() {
		Convey("简单正则表达式", func() {
			q := &RegexpQuery{
				Field: "name",
				Value: "test.*",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex":   "test.*",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("数字匹配正则", func() {
			q := &RegexpQuery{
				Field: "phone",
				Value: "\\d{11}",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"phone": map[string]interface{}{
					"$regex":   "\\d{11}",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("邮箱匹配正则", func() {
			q := &RegexpQuery{
				Field: "email",
				Value: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"email": map[string]interface{}{
					"$regex":   "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("字符类正则", func() {
			q := &RegexpQuery{
				Field: "code",
				Value: "[A-Z]{2}[0-9]{4}",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"code": map[string]interface{}{
					"$regex":   "[A-Z]{2}[0-9]{4}",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("空正则表达式", func() {
			q := &RegexpQuery{
				Field: "empty",
				Value: "",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"empty": map[string]interface{}{
					"$regex":   "",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("OR 选择正则", func() {
			q := &RegexpQuery{
				Field: "status",
				Value: "active|pending|completed",
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"status": map[string]interface{}{
					"$regex":   "active|pending|completed",
					"$options": "i",
				},
			}
			So(result, ShouldResemble, expected)
		})
	})
}