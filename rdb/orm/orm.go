package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/rdb/query"
)

// Repository 泛型仓库接口，T 为实体类型
type Repository[T any] interface {
	// 自动迁移表结构
	AutoMigrate(ctx context.Context) error

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

// repositoryImpl Repository 接口的实现
type repositoryImpl[T any] struct {
	db    database.Database
	table string
	model *database.TableModel
}

// NewRepository 创建新的 Repository 实例
func NewRepository[T any](db database.Database) (Repository[T], error) {
	var zero T

	// 使用 TableModelBuilder 从结构体构建模型
	builder := database.NewTableModelBuilder()
	model, err := builder.FromStruct(zero)
	if err != nil {
		return nil, fmt.Errorf("failed to build table model: %w", err)
	}

	repo := &repositoryImpl[T]{
		db:    db,
		table: model.Table,
		model: model,
	}

	return repo, nil
}

// AutoMigrate 自动迁移表结构
func (r *repositoryImpl[T]) AutoMigrate(ctx context.Context) error {
	return r.db.Migrate(ctx, r.model)
}

// Create 创建记录
func (r *repositoryImpl[T]) Create(ctx context.Context, entity *T, opts ...database.CreateOption) error {
	builder := r.db.GetBuilder()
	record := builder.FromStruct(entity)
	return r.db.Create(ctx, r.table, record, opts...)
}

// Get 根据主键获取记录
func (r *repositoryImpl[T]) Get(ctx context.Context, id any) (*T, error) {
	pk := r.buildPrimaryKey(id)
	record, err := r.db.Get(ctx, r.table, pk)
	if err != nil {
		return nil, err
	}

	var entity T
	if err := record.ScanStruct(&entity); err != nil {
		return nil, fmt.Errorf("failed to scan result: %w", err)
	}

	return &entity, nil
}

// Update 更新记录
func (r *repositoryImpl[T]) Update(ctx context.Context, entity *T) error {
	// 从实体中提取主键
	pk := r.extractPrimaryKey(entity)
	if len(pk) == 0 {
		return fmt.Errorf("primary key not found in entity")
	}

	builder := r.db.GetBuilder()
	record := builder.FromStruct(entity)
	return r.db.Update(ctx, r.table, pk, record)
}

// Delete 根据主键删除记录
func (r *repositoryImpl[T]) Delete(ctx context.Context, id any) error {
	pk := r.buildPrimaryKey(id)
	return r.db.Delete(ctx, r.table, pk)
}

// Find 根据查询条件查询多条记录
func (r *repositoryImpl[T]) Find(ctx context.Context, q query.Query, opts ...database.QueryOption) ([]*T, error) {
	records, err := r.db.Find(ctx, r.table, q, opts...)
	if err != nil {
		return nil, err
	}

	var entities []*T
	for _, record := range records {
		var entity T
		if err := record.ScanStruct(&entity); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		entities = append(entities, &entity)
	}

	return entities, nil
}

// FindOne 根据查询条件查询单条记录
func (r *repositoryImpl[T]) FindOne(ctx context.Context, q query.Query) (*T, error) {
	opts := []database.QueryOption{
		func(options *database.QueryOptions) {
			options.Limit = 1
		},
	}

	entities, err := r.Find(ctx, q, opts...)
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		return nil, database.ErrRecordNotFound
	}

	return entities[0], nil
}

// Count 统计记录数量
func (r *repositoryImpl[T]) Count(ctx context.Context, q query.Query) (int64, error) {
	// 这里需要使用聚合查询来统计数量
	// 简化实现：先查询所有记录然后计数
	records, err := r.db.Find(ctx, r.table, q)
	if err != nil {
		return 0, err
	}
	return int64(len(records)), nil
}

// Exists 检查记录是否存在
func (r *repositoryImpl[T]) Exists(ctx context.Context, q query.Query) (bool, error) {
	count, err := r.Count(ctx, q)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BatchCreate 批量创建记录
func (r *repositoryImpl[T]) BatchCreate(ctx context.Context, entities []*T, opts ...database.CreateOption) error {
	builder := r.db.GetBuilder()
	var records []database.Record

	for _, entity := range entities {
		record := builder.FromStruct(entity)
		records = append(records, record)
	}

	return r.db.BatchCreate(ctx, r.table, records, opts...)
}

// BatchUpdate 批量更新记录
func (r *repositoryImpl[T]) BatchUpdate(ctx context.Context, entities []*T) error {
	builder := r.db.GetBuilder()
	var pks []map[string]any
	var records []database.Record

	for _, entity := range entities {
		pk := r.extractPrimaryKey(entity)
		if len(pk) == 0 {
			return fmt.Errorf("primary key not found in entity")
		}

		pks = append(pks, pk)
		record := builder.FromStruct(entity)
		records = append(records, record)
	}

	return r.db.BatchUpdate(ctx, r.table, pks, records)
}

// BatchDelete 批量删除记录
func (r *repositoryImpl[T]) BatchDelete(ctx context.Context, ids []any) error {
	var pks []map[string]any

	for _, id := range ids {
		pk := r.buildPrimaryKey(id)
		pks = append(pks, pk)
	}

	return r.db.BatchDelete(ctx, r.table, pks)
}

// buildPrimaryKey 构建主键映射
func (r *repositoryImpl[T]) buildPrimaryKey(id any) map[string]any {
	pk := make(map[string]any)

	if len(r.model.PrimaryKey) == 1 {
		// 单一主键
		pk[r.model.PrimaryKey[0]] = id
	} else if len(r.model.PrimaryKey) > 1 {
		// 复合主键，id 应该是一个 map 或 struct
		if idMap, ok := id.(map[string]any); ok {
			for _, keyName := range r.model.PrimaryKey {
				if value, exists := idMap[keyName]; exists {
					pk[keyName] = value
				}
			}
		}
	}

	return pk
}

// extractPrimaryKey 从实体中提取主键
func (r *repositoryImpl[T]) extractPrimaryKey(entity *T) map[string]any {
	pk := make(map[string]any)

	rv := reflect.ValueOf(entity)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()

	// 遍历主键字段
	for _, keyName := range r.model.PrimaryKey {
		// 查找对应的结构体字段
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			fieldName := r.getFieldName(field)

			if fieldName == keyName {
				value := rv.Field(i).Interface()
				pk[keyName] = value
				break
			}
		}
	}

	return pk
}

// getFieldName 获取字段的数据库列名
func (r *repositoryImpl[T]) getFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("rdb")
	if tag == "" || tag == "-" {
		return field.Name
	}

	// 解析第一部分作为字段名
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" && !strings.Contains(parts[0], "=") {
		return parts[0]
	}

	return field.Name
}
