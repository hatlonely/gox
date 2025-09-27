package rdb

import (
	"context"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrDuplicateKey     = errors.New("duplicate key")
	ErrInvalidCondition = errors.New("invalid condition")
)

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
	Scan(dest any) error
	ScanStruct(dest any) error

	// 写入时的数据提取方法
	Fields() map[string]any
}

// RecordBuilder 记录构建器，用于创建Record实例
type RecordBuilder interface {
	FromStruct(v any) Record
	FromMap(data map[string]any, table string) Record
}

// Transaction 事务接口，继承RDB的所有功能
type Transaction interface {
	RDB
	Commit() error
	Rollback() error
}

// RDB ORM接口，统一使用Record接口实现类型灵活性
type RDB interface {
	// Migrate 自动创建/更新表结构
	Migrate(ctx context.Context, table string, model *TableModel) error

	// Create 创建记录
	Create(ctx context.Context, table string, record Record, opts ...CreateOption) error

	// Get 根据主键获取记录
	Get(ctx context.Context, table string, pk map[string]any) (Record, error)

	// Update 更新记录（根据主键）
	Update(ctx context.Context, table string, pk map[string]any, record Record) error

	// Delete 根据主键删除记录
	Delete(ctx context.Context, table string, pk map[string]any) error

	// Find 根据查询条件查询多条记录
	Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error)

	// Aggregate 执行聚合查询
	Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error)

	// BatchCreate 批量创建记录
	BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error

	// BatchUpdate 批量更新记录
	BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error

	// BatchDelete 批量删除记录
	BatchDelete(ctx context.Context, table string, pks []map[string]any) error

	// BeginTx 开始事务
	BeginTx(ctx context.Context) (Transaction, error)

	// WithTx 在事务中执行操作
	WithTx(ctx context.Context, fn func(tx Transaction) error) error

	// GetBuilder 获取记录构建器
	GetBuilder() RecordBuilder

	// Close 关闭连接
	Close() error
}

// 工厂方法
func NewRDBWithOptions(options *ref.TypeOptions) (RDB, error) {
	return nil, errors.New("not implemented yet")
}
