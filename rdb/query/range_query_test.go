package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRangeQueryType(t *testing.T) {
	Convey("测试 RangeQuery Type 方法", t, func() {
		q := &RangeQuery{Field: "age"}
		So(q.Type(), ShouldEqual, QueryTypeRange)
	})
}

func TestRangeQueryToES(t *testing.T) {
	Convey("测试 RangeQuery ToES 方法", t, func() {
		Convey("只有 Gt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gt:    18,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"range": map[string]interface{}{
					"age": map[string]interface{}{
						"gt": 18,
					},
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Gte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"range": map[string]interface{}{
					"age": map[string]interface{}{
						"gte": 18,
					},
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Lt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lt:    65,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"range": map[string]interface{}{
					"age": map[string]interface{}{
						"lt": 65,
					},
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Lte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lte:   65,
			}
			result := q.ToES()
			expected := map[string]interface{}{
				"range": map[string]interface{}{
					"age": map[string]interface{}{
						"lte": 65,
					},
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("多个条件组合", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
				Lt:    65,
			}
			result := q.ToES()
			rangeQuery := result["range"].(map[string]interface{})["age"].(map[string]interface{})
			So(rangeQuery["gte"], ShouldEqual, 18)
			So(rangeQuery["lt"], ShouldEqual, 65)
		})

		Convey("包含额外字段", func() {
			q := &RangeQuery{
				Field: "timestamp",
				Gte:   "2023-01-01",
				Extra: map[string]interface{}{
					"format": "yyyy-MM-dd",
					"time_zone": "+08:00",
				},
			}
			result := q.ToES()
			rangeQuery := result["range"].(map[string]interface{})["timestamp"].(map[string]interface{})
			So(rangeQuery["gte"], ShouldEqual, "2023-01-01")
			So(rangeQuery["format"], ShouldEqual, "yyyy-MM-dd")
			So(rangeQuery["time_zone"], ShouldEqual, "+08:00")
		})
	})
}

func TestRangeQueryToSQL(t *testing.T) {
	Convey("测试 RangeQuery ToSQL 方法", t, func() {
		Convey("只有 Gt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gt:    18,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age > ?")
			So(args, ShouldResemble, []interface{}{18})
		})

		Convey("只有 Gte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age >= ?")
			So(args, ShouldResemble, []interface{}{18})
		})

		Convey("只有 Lt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lt:    65,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age < ?")
			So(args, ShouldResemble, []interface{}{65})
		})

		Convey("只有 Lte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lte:   65,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age <= ?")
			So(args, ShouldResemble, []interface{}{65})
		})

		Convey("多个条件组合", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
				Lt:    65,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "age >= ? AND age < ?")
			So(args, ShouldResemble, []interface{}{18, 65})
		})

		Convey("所有条件组合", func() {
			q := &RangeQuery{
				Field: "score",
				Gt:    0,
				Gte:   10,
				Lt:    90,
				Lte:   100,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "score > ? AND score >= ? AND score < ? AND score <= ?")
			So(args, ShouldResemble, []interface{}{0, 10, 90, 100})
		})

		Convey("无条件", func() {
			q := &RangeQuery{Field: "age"}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "1=1")
			So(args, ShouldBeEmpty)
		})
	})
}

func TestRangeQueryToMongo(t *testing.T) {
	Convey("测试 RangeQuery ToMongo 方法", t, func() {
		Convey("只有 Gt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gt:    18,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{
					"$gt": 18,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Gte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": 18,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Lt 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lt:    65,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{
					"$lt": 65,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("只有 Lte 条件", func() {
			q := &RangeQuery{
				Field: "age",
				Lte:   65,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{
					"$lte": 65,
				},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("多个条件组合", func() {
			q := &RangeQuery{
				Field: "age",
				Gte:   18,
				Lt:    65,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			ageCondition := result["age"].(map[string]interface{})
			So(ageCondition["$gte"], ShouldEqual, 18)
			So(ageCondition["$lt"], ShouldEqual, 65)
		})

		Convey("所有条件组合", func() {
			q := &RangeQuery{
				Field: "score",
				Gt:    0,
				Gte:   10,
				Lt:    90,
				Lte:   100,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			scoreCondition := result["score"].(map[string]interface{})
			So(scoreCondition["$gt"], ShouldEqual, 0)
			So(scoreCondition["$gte"], ShouldEqual, 10)
			So(scoreCondition["$lt"], ShouldEqual, 90)
			So(scoreCondition["$lte"], ShouldEqual, 100)
		})

		Convey("无条件", func() {
			q := &RangeQuery{Field: "age"}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			expected := map[string]interface{}{
				"age": map[string]interface{}{},
			}
			So(result, ShouldResemble, expected)
		})
	})
}