package orm

import (
	"context"

	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/rdb/query"
)

// Repository 泛型仓库接口，T 为实体类型
type Repository[T any] interface {
	// 基础 CRUD 操作
	Create(ctx context.Context, entity *T, opts ...database.CreateOption) error
	Get(ctx context.Context, id any) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id any) error

	// 查询操作
	Find(ctx context.Context, q query.Query, opts ...database.QueryOption) ([]*T, error)
	FindOne(ctx context.Context, q query.Query) (*T, error)
	Count(ctx context.Context, q query.Query) (int64, error)
	Exists(ctx context.Context, q query.Query) (bool, error)

	// 批量操作
	BatchCreate(ctx context.Context, entities []*T, opts ...database.CreateOption) error
	BatchUpdate(ctx context.Context, entities []*T) error
	BatchDelete(ctx context.Context, ids []any) error
}
