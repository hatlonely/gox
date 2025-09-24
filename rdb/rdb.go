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

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
}

// RDB ORM接口，支持结构体到数据库的直接映射
type RDB[T any] interface {
	// Migrate 自动创建/更新表结构
	Migrate(ctx context.Context) error
	
	// Create 创建记录
	Create(ctx context.Context, record *T, opts ...CreateOption) error
	
	// Get 根据主键获取记录
	Get(ctx context.Context, id any) (*T, error)
	
	// Update 更新记录（根据主键）
	Update(ctx context.Context, record *T) error
	
	// Delete 根据主键删除记录
	Delete(ctx context.Context, id any) error
	
	// Find 根据查询条件查询多条记录
	Find(ctx context.Context, query QueryNode, opts ...QueryOption) ([]*T, error)
	
	// FindOne 根据查询条件查询单条记录
	FindOne(ctx context.Context, query QueryNode) (*T, error)
	
	// Count 统计记录数量
	Count(ctx context.Context, query QueryNode) (int64, error)
	
	// BatchCreate 批量创建记录
	BatchCreate(ctx context.Context, records []*T, opts ...CreateOption) error
	
	// BatchUpdate 批量更新记录
	BatchUpdate(ctx context.Context, records []*T) error
	
	// BatchDelete 批量删除记录
	BatchDelete(ctx context.Context, ids []any) error
	
	// BeginTx 开始事务
	BeginTx(ctx context.Context) (RDB[T], error)
	
	// WithTx 在事务中执行操作
	WithTx(ctx context.Context, fn func(tx RDB[T]) error) error
	
	// Close 关闭连接
	Close() error
}

// 工厂方法
func NewRDBWithOptions[T any](options *ref.TypeOptions) (RDB[T], error) {
	return nil, errors.New("not implemented yet")
}
