package rdb

import (
	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/ref"
)

// NewDatabaseWithOptions 使用指定配置创建数据库实例
func NewDatabaseWithOptions(options *ref.TypeOptions) (database.Database, error) {
	return database.NewDatabaseWithOptions(options)
}