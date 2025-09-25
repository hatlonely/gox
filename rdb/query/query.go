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

// Query 查询节点接口
type Query interface {
	Type() QueryType
	// 后端适配器接口
	ToES() map[string]interface{}
	ToSQL() (string, []interface{}, error)
	ToMongo() (map[string]interface{}, error)
}

