package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueryType(t *testing.T) {
	Convey("测试 QueryType 常量", t, func() {
		So(QueryTypeBool, ShouldEqual, QueryType("bool"))
		So(QueryTypeTerm, ShouldEqual, QueryType("term"))
		So(QueryTypeMatch, ShouldEqual, QueryType("match"))
		So(QueryTypeRange, ShouldEqual, QueryType("range"))
		So(QueryTypeExists, ShouldEqual, QueryType("exists"))
		So(QueryTypeWildcard, ShouldEqual, QueryType("wildcard"))
		So(QueryTypePrefix, ShouldEqual, QueryType("prefix"))
		So(QueryTypeRegexp, ShouldEqual, QueryType("regexp"))
	})
}