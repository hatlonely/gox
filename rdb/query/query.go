package query

// QueryType 查询类型
type QueryType string

const (
	QueryTypeBool     QueryType = "bool"
	QueryTypeTerm     QueryType = "term"
	QueryTypeMatch    QueryType = "match"
	QueryTypeRange    QueryType = "range"
	QueryTypeExists   QueryType = "exists"
	QueryTypeWildcard QueryType = "wildcard"
	QueryTypePrefix   QueryType = "prefix"
	QueryTypeRegexp   QueryType = "regexp"
)

// QueryNode 查询节点接口
type QueryNode interface {
	Type() QueryType
	Children() []QueryNode
	// 后端适配器接口
	ToES() map[string]interface{}
	ToSQL() (string, []interface{}, error)
	ToMongo() (map[string]interface{}, error)
}

// BoolQuery 布尔查询
type BoolQuery struct {
	Must           []QueryNode `json:"must,omitempty"`
	Should         []QueryNode `json:"should,omitempty"`
	MustNot        []QueryNode `json:"must_not,omitempty"`
	Filter         []QueryNode `json:"filter,omitempty"`
	MinShouldMatch *int        `json:"minimum_should_match,omitempty"`
}

// TermQuery 精确匹配查询
type TermQuery struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// RangeQuery 范围查询
type RangeQuery struct {
	Field string                 `json:"field"`
	Gt    interface{}            `json:"gt,omitempty"`
	Gte   interface{}            `json:"gte,omitempty"`
	Lt    interface{}            `json:"lt,omitempty"`
	Lte   interface{}            `json:"lte,omitempty"`
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// MatchQuery 全文搜索查询
type MatchQuery struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// ExistsQuery 字段存在查询
type ExistsQuery struct {
	Field string `json:"field"`
}
