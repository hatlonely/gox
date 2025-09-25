package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBoolQueryType(t *testing.T) {
	Convey("测试 BoolQuery Type 方法", t, func() {
		q := &BoolQuery{}
		So(q.Type(), ShouldEqual, QueryTypeBool)
	})
}

func TestBoolQueryToES(t *testing.T) {
	Convey("测试 BoolQuery ToES 方法", t, func() {
		Convey("空的 BoolQuery", func() {
			q := &BoolQuery{}
			result := q.ToES()
			expected := map[string]interface{}{
				"bool": map[string]interface{}{},
			}
			So(result, ShouldResemble, expected)
		})

		Convey("包含 Must 条件", func() {
			q := &BoolQuery{
				Must: []Query{
					&TermQuery{Field: "status", Value: "active"},
				},
			}
			result := q.ToES()
			So(result["bool"].(map[string]interface{})["must"], ShouldHaveLength, 1)
		})

		Convey("包含 Should 条件", func() {
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "category", Value: "tech"},
				},
			}
			result := q.ToES()
			So(result["bool"].(map[string]interface{})["should"], ShouldHaveLength, 1)
		})

		Convey("包含 MustNot 条件", func() {
			q := &BoolQuery{
				MustNot: []Query{
					&TermQuery{Field: "deleted", Value: true},
				},
			}
			result := q.ToES()
			So(result["bool"].(map[string]interface{})["must_not"], ShouldHaveLength, 1)
		})

		Convey("包含 Filter 条件", func() {
			q := &BoolQuery{
				Filter: []Query{
					&RangeQuery{Field: "timestamp", Gte: 1000},
				},
			}
			result := q.ToES()
			So(result["bool"].(map[string]interface{})["filter"], ShouldHaveLength, 1)
		})

		Convey("包含 MinShouldMatch", func() {
			minMatch := 2
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "tag1", Value: "value1"},
					&TermQuery{Field: "tag2", Value: "value2"},
				},
				MinShouldMatch: &minMatch,
			}
			result := q.ToES()
			So(result["bool"].(map[string]interface{})["minimum_should_match"], ShouldEqual, 2)
		})
	})
}

func TestBoolQueryToSQL(t *testing.T) {
	Convey("测试 BoolQuery ToSQL 方法", t, func() {
		Convey("空的 BoolQuery", func() {
			q := &BoolQuery{}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "1=1")
			So(args, ShouldBeEmpty)
		})

		Convey("包含 Must 条件", func() {
			q := &BoolQuery{
				Must: []Query{
					&TermQuery{Field: "status", Value: "active"},
				},
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "(status = ?)")
			So(args, ShouldResemble, []interface{}{"active"})
		})

		Convey("包含 Should 条件", func() {
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "category", Value: "tech"},
					&TermQuery{Field: "category", Value: "science"},
				},
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "(category = ? OR category = ?)")
			So(args, ShouldResemble, []interface{}{"tech", "science"})
		})

		Convey("包含 Should 条件和 MinShouldMatch", func() {
			minMatch := 2
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "tag1", Value: "value1"},
					&TermQuery{Field: "tag2", Value: "value2"},
					&TermQuery{Field: "tag3", Value: "value3"},
				},
				MinShouldMatch: &minMatch,
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldContainSubstring, "CASE WHEN")
			So(sql, ShouldContainSubstring, ">= 2")
			So(args, ShouldHaveLength, 3)
		})

		Convey("包含 MustNot 条件", func() {
			q := &BoolQuery{
				MustNot: []Query{
					&TermQuery{Field: "deleted", Value: true},
				},
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldEqual, "(NOT (deleted = ?))")
			So(args, ShouldResemble, []interface{}{true})
		})

		Convey("复合条件", func() {
			q := &BoolQuery{
				Must: []Query{
					&TermQuery{Field: "status", Value: "active"},
				},
				Filter: []Query{
					&RangeQuery{Field: "age", Gte: 18},
				},
			}
			sql, args, err := q.ToSQL()
			So(err, ShouldBeNil)
			So(sql, ShouldContainSubstring, "status = ?")
			So(sql, ShouldContainSubstring, "age >= ?")
			So(sql, ShouldContainSubstring, " AND ")
			So(args, ShouldResemble, []interface{}{"active", 18})
		})
	})
}

func TestBoolQueryToMongo(t *testing.T) {
	Convey("测试 BoolQuery ToMongo 方法", t, func() {
		Convey("空的 BoolQuery", func() {
			q := &BoolQuery{}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result, ShouldResemble, map[string]interface{}{})
		})

		Convey("包含 Must 条件", func() {
			q := &BoolQuery{
				Must: []Query{
					&TermQuery{Field: "status", Value: "active"},
				},
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result, ShouldResemble, map[string]interface{}{
				"status": "active",
			})
		})

		Convey("包含 Should 条件", func() {
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "category", Value: "tech"},
					&TermQuery{Field: "category", Value: "science"},
				},
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result["$or"], ShouldHaveLength, 2)
		})

		Convey("包含 Should 条件和 MinShouldMatch", func() {
			minMatch := 2
			q := &BoolQuery{
				Should: []Query{
					&TermQuery{Field: "tag1", Value: "value1"},
					&TermQuery{Field: "tag2", Value: "value2"},
					&TermQuery{Field: "tag3", Value: "value3"},
				},
				MinShouldMatch: &minMatch,
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result["$expr"], ShouldNotBeNil)
		})

		Convey("包含 MustNot 条件", func() {
			q := &BoolQuery{
				MustNot: []Query{
					&TermQuery{Field: "deleted", Value: true},
				},
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result["$nor"], ShouldHaveLength, 1)
		})

		Convey("复合条件", func() {
			q := &BoolQuery{
				Must: []Query{
					&TermQuery{Field: "status", Value: "active"},
				},
				Filter: []Query{
					&RangeQuery{Field: "age", Gte: 18},
				},
			}
			result, err := q.ToMongo()
			So(err, ShouldBeNil)
			So(result["$and"], ShouldHaveLength, 2)
		})
	})
}