package rdb

import (
	"context"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrDuplicateKey     = errors.New("duplicate key")
	ErrInvalidCondition = errors.New("invalid condition")
)

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
	MustClauses    []QueryNode `json:"must,omitempty"`
	ShouldClauses  []QueryNode `json:"should,omitempty"`
	MustNotClauses []QueryNode `json:"must_not,omitempty"`
	FilterClauses  []QueryNode `json:"filter,omitempty"`
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

// CreateOptions 创建记录时的选项
type CreateOptions struct {
	IgnoreConflict   bool
	UpdateOnConflict bool
}

type CreateOption func(*CreateOptions)

// QueryOptions 查询选项
type QueryOptions struct {
	Limit     int
	Offset    int
	OrderBy   string
	OrderDesc bool
}

type QueryOption func(*QueryOptions)

// Record 通用记录接口，用于数据转换
type Record interface {
	// 查询时的转换方法
	Scan(dest interface{}) error
	ScanStruct(dest interface{}) error
	
	// 写入时的数据提取方法  
	Fields() map[string]interface{}
	TableName() string
	PrimaryKey() (string, interface{})
}

// RecordBuilder 记录构建器，用于创建Record实例
type RecordBuilder interface {
	FromStruct(v interface{}) Record
	FromMap(data map[string]interface{}, table string) Record
}

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
}

// RDB ORM接口，统一使用Record接口实现类型灵活性
type RDB interface {
	// Migrate 自动创建/更新表结构
	Migrate(ctx context.Context, table string, model interface{}) error

	// Create 创建记录
	Create(ctx context.Context, record Record, opts ...CreateOption) error

	// Get 根据主键获取记录
	Get(ctx context.Context, table string, id any) (Record, error)

	// Update 更新记录（根据主键）
	Update(ctx context.Context, record Record) error

	// Delete 根据主键删除记录
	Delete(ctx context.Context, table string, id any) error

	// Find 根据查询条件查询多条记录
	Find(ctx context.Context, table string, query QueryNode, opts ...QueryOption) ([]Record, error)

	// FindOne 根据查询条件查询单条记录
	FindOne(ctx context.Context, table string, query QueryNode) (Record, error)

	// Count 统计记录数量
	Count(ctx context.Context, table string, query QueryNode) (int64, error)

	// BatchCreate 批量创建记录
	BatchCreate(ctx context.Context, records []Record, opts ...CreateOption) error

	// BatchUpdate 批量更新记录
	BatchUpdate(ctx context.Context, records []Record) error

	// BatchDelete 批量删除记录
	BatchDelete(ctx context.Context, table string, ids []any) error

	// BeginTx 开始事务
	BeginTx(ctx context.Context) (RDB, error)

	// WithTx 在事务中执行操作
	WithTx(ctx context.Context, fn func(tx RDB) error) error

	// GetBuilder 获取记录构建器
	GetBuilder() RecordBuilder

	// Close 关闭连接
	Close() error
}

// 工厂方法
func NewRDBWithOptions(options *ref.TypeOptions) (RDB, error) {
	return nil, errors.New("not implemented yet")
}
