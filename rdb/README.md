# RDB - 关系数据库 ORM 包

统一的数据库 ORM 接口，支持多种数据库后端。

## 支持的数据库

- **SQL**: MySQL, SQLite
- **MongoDB**: NoSQL 文档数据库
- **Elasticsearch**: 搜索引擎数据库

## 快速开始

```go
import (
    "github.com/hatlonely/gox/rdb"
    "github.com/hatlonely/gox/ref"
)

// 创建 MySQL 数据库
db, err := rdb.NewDatabaseWithOptions(&ref.TypeOptions{
    Namespace: "github.com/hatlonely/gox/rdb/database",
    Type:      "SQL",
    Options: &database.SQLOptions{
        Driver:   "mysql",
        Host:     "localhost",
        Port:     "3306",
        Database: "mydb",
        Username: "user",
        Password: "pass",
    },
})

// 创建 MongoDB 数据库
db, err := rdb.NewDatabaseWithOptions(&ref.TypeOptions{
    Namespace: "github.com/hatlonely/gox/rdb/database",
    Type:      "Mongo",
    Options: &database.MongoOptions{
        Host:     "localhost",
        Port:     27017,
        Database: "mydb",
    },
})
```

## 基本操作

```go
// 创建记录
err = db.Create(ctx, "users", record)

// 查询记录
record, err := db.Get(ctx, "users", map[string]any{"id": 1})

// 更新记录
err = db.Update(ctx, "users", map[string]any{"id": 1}, record)

// 删除记录
err = db.Delete(ctx, "users", map[string]any{"id": 1})

// 条件查询
records, err := db.Find(ctx, "users", query.Eq("status", "active"))

// 事务操作
err = db.WithTx(ctx, func(tx database.Transaction) error {
    // 在事务中执行操作
    return tx.Create(ctx, "users", record)
})
```

## 配置示例

### MySQL 配置
```go
&database.SQLOptions{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     "3306", 
    Database: "mydb",
    Username: "user",
    Password: "pass",
    Charset:  "utf8mb4",
    MaxConns: 10,
    MaxIdle:  5,
}
```

### MongoDB 配置
```go
&database.MongoOptions{
    Host:        "localhost",
    Port:        27017,
    Database:    "mydb",
    Username:    "user",
    Password:    "pass",
    AuthSource:  "admin",
    Timeout:     30 * time.Second,
    MaxPoolSize: 100,
}
```

### Elasticsearch 配置
```go
&database.ESOptions{
    Addresses:  []string{"http://localhost:9200"},
    Username:   "user",
    Password:   "pass",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
}
```