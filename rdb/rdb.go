package rdb

import (
	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/rdb/repository"
	"github.com/hatlonely/gox/ref"
)

// NewDatabaseWithOptions 使用指定配置创建数据库实例
func NewDatabaseWithOptions(options *ref.TypeOptions) (database.Database, error) {
	return database.NewDatabaseWithOptions(options)
}

// NewRepository 创建新的 Repository 实例
func NewRepository[T any](db database.Database) (repository.Repository[T], error) {
	return repository.NewRepository[T](db)
}