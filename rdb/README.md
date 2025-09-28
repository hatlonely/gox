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

### 使用 Database 接口（底层操作）

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

### 使用 Repository（推荐方式）

```go
// 定义实体结构
type User struct {
    ID     int    `rdb:"id,primary_key,auto_increment"`
    Name   string `rdb:"name"`
    Email  string `rdb:"email,unique"`
    Status string `rdb:"status,default=active"`
}

// 创建 Repository
userRepo, err := rdb.NewRepository[User](db)

// 自动迁移表结构
err = userRepo.Migrate(ctx)

// CRUD 操作
user := &User{Name: "John", Email: "john@example.com"}
err = userRepo.Create(ctx, user)

// 根据 ID 查询
user, err := userRepo.Get(ctx, 1)

// 条件查询
users, err := userRepo.Find(ctx, query.Eq("status", "active"))

// 查询单条记录
user, err := userRepo.FindOne(ctx, query.Eq("email", "john@example.com"))

// 更新记录
user.Status = "inactive"
err = userRepo.Update(ctx, user)

// 删除记录
err = userRepo.Delete(ctx, 1)

// 批量操作
users := []*User{{Name: "Alice"}, {Name: "Bob"}}
err = userRepo.BatchCreate(ctx, users)

// 统计和检查
count, err := userRepo.Count(ctx, query.Eq("status", "active"))
exists, err := userRepo.Exists(ctx, query.Eq("email", "john@example.com"))
```

## 实体标签说明

Repository 使用结构体标签来定义表结构：

```go
type User struct {
    ID        int       `rdb:"id,primary_key,auto_increment"`
    Name      string    `rdb:"name,not_null"`
    Email     string    `rdb:"email,unique,index"`
    Status    string    `rdb:"status,default=active"`
    CreatedAt time.Time `rdb:"created_at,default=CURRENT_TIMESTAMP"`
    UpdatedAt time.Time `rdb:"updated_at,on_update=CURRENT_TIMESTAMP"`
}
```

支持的标签选项：
- `primary_key`: 主键
- `auto_increment`: 自增
- `not_null`: 非空
- `unique`: 唯一约束
- `index`: 创建索引
- `default=value`: 默认值
- `on_update=value`: 更新时的值

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