package orm

import (
	"context"
	"testing"
	"time"

	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/rdb/query"
	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的实体结构体
type User struct {
	ID       int       `rdb:"id,primary"`
	Name     string    `rdb:"name,required"`
	Email    string    `rdb:"email,unique"`
	Age      int       `rdb:"age"`
	Active   bool      `rdb:"active"`
	Score    float64   `rdb:"score"`
	CreateAt time.Time `rdb:"create_at"`
}

// 设置表名
func (u User) TableName() string {
	return "users"
}

// 复合主键测试实体
type UserProfile struct {
	UserID   int    `rdb:"user_id,primary"`
	Platform string `rdb:"platform,primary"`
	Username string `rdb:"username,required"`
	Avatar   string `rdb:"avatar"`
}

func (up UserProfile) TableName() string {
	return "user_profiles"
}

// 测试配置 - 复用 mysql_test.go 中的配置
var testMySQLOptions = &database.SQLOptions{
	Driver:   "mysql",
	Host:     "localhost",
	Port:     "3306",
	Database: "testdb",
	Username: "testuser",
	Password: "testpass",
	Charset:  "utf8mb4",
	MaxConns: 10,
	MaxIdle:  5,
}

func TestNewRepository(t *testing.T) {
	Convey("测试 NewRepository 方法", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		Convey("创建 User Repository", func() {
			repo, err := NewRepository[User](db)
			So(err, ShouldBeNil)
			So(repo, ShouldNotBeNil)

			// 检查内部状态
			impl := repo.(*repositoryImpl[User])
			So(impl.db, ShouldEqual, db)
			So(impl.table, ShouldEqual, "User") // 默认使用结构体名
			So(impl.model, ShouldNotBeNil)
			So(len(impl.model.PrimaryKey), ShouldEqual, 1)
			So(impl.model.PrimaryKey[0], ShouldEqual, "id")
		})

		Convey("创建复合主键 Repository", func() {
			repo, err := NewRepository[UserProfile](db)
			So(err, ShouldBeNil)
			So(repo, ShouldNotBeNil)

			impl := repo.(*repositoryImpl[UserProfile])
			So(len(impl.model.PrimaryKey), ShouldEqual, 2)
			So(impl.model.PrimaryKey, ShouldContain, "user_id")
			So(impl.model.PrimaryKey, ShouldContain, "platform")
		})
	})
}

func TestRepositoryMigrate(t *testing.T) {
	Convey("测试 Repository Migrate 方法", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[User](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "User")

		Convey("迁移表结构", func() {
			err := repo.Migrate(ctx)
			So(err, ShouldBeNil)

			// 验证表是否创建成功 - 尝试创建一个用户
			user := &User{
				ID:       1,
				Name:     "Test User",
				Email:    "test@example.com",
				Age:      25,
				Active:   true,
				Score:    88.5,
				CreateAt: time.Now(),
			}
			err = repo.Create(ctx, user)
			So(err, ShouldBeNil)
		})

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "User")
		})
	})
}

func TestRepositoryCRUD(t *testing.T) {
	Convey("测试 Repository CRUD 操作", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[User](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "User")

		// 迁移表结构
		err = repo.Migrate(ctx)
		So(err, ShouldBeNil)

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "User")
		})

		Convey("Create 操作", func() {
			user := &User{
				ID:       1000, // 使用唯一 ID
				Name:     "John Doe",
				Email:    "john1000@example.com",
				Age:      30,
				Active:   true,
				Score:    95.5,
				CreateAt: time.Now(),
			}

			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)
		})

		Convey("Create 和 Get 操作", func() {
			user := &User{
				ID:       1001,
				Name:     "Jane Smith",
				Email:    "jane1001@example.com",
				Age:      28,
				Active:   false,
				Score:    87.3,
				CreateAt: time.Now(),
			}

			// 创建用户
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)

			// 获取用户
			retrievedUser, err := repo.Get(ctx, 1001)
			So(err, ShouldBeNil)
			So(retrievedUser, ShouldNotBeNil)
			So(retrievedUser.ID, ShouldEqual, 1001)
			So(retrievedUser.Name, ShouldEqual, "Jane Smith")
			So(retrievedUser.Email, ShouldEqual, "jane1001@example.com")
			So(retrievedUser.Age, ShouldEqual, 28)
			So(retrievedUser.Active, ShouldEqual, false)
			So(retrievedUser.Score, ShouldAlmostEqual, 87.3, 0.01) // 使用 AlmostEqual 处理浮点精度
		})

		Convey("Update 操作", func() {
			user := &User{
				ID:       1002,
				Name:     "Bob Wilson",
				Email:    "bob1002@example.com",
				Age:      35,
				Active:   true,
				Score:    92.1,
				CreateAt: time.Now(),
			}

			// 创建用户
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)

			// 更新用户信息
			user.Name = "Bob Wilson Updated"
			user.Age = 36
			user.Score = 93.5
			err = repo.Update(ctx, user)
			So(err, ShouldBeNil)

			// 验证更新结果
			updatedUser, err := repo.Get(ctx, 1002)
			So(err, ShouldBeNil)
			So(updatedUser.Name, ShouldEqual, "Bob Wilson Updated")
			So(updatedUser.Age, ShouldEqual, 36)
			So(updatedUser.Score, ShouldEqual, 93.5)
		})

		Convey("Delete 操作", func() {
			user := &User{
				ID:       1003,
				Name:     "Alice Brown",
				Email:    "alice3@example.com",
				Age:      27,
				Active:   true,
				Score:    89.7,
				CreateAt: time.Now(),
			}

			// 创建用户
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)

			// 验证用户存在
			_, err = repo.Get(ctx, 1003)
			So(err, ShouldBeNil)

			// 删除用户
			err = repo.Delete(ctx, 1003)
			So(err, ShouldBeNil)

			// 验证用户已被删除
			_, err = repo.Get(ctx, 1003)
			So(err, ShouldEqual, database.ErrRecordNotFound)
		})
	})
}

func TestRepositoryQuery(t *testing.T) {
	Convey("测试 Repository 查询操作", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[User](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "User")

		// 迁移表结构
		err = repo.Migrate(ctx)
		So(err, ShouldBeNil)

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "User")
		})

		// 准备测试数据
		users := []*User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 25, Active: true, Score: 85.5},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, Active: false, Score: 90.2},
			{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35, Active: true, Score: 78.8},
			{ID: 4, Name: "Diana", Email: "diana@example.com", Age: 28, Active: true, Score: 92.1},
		}

		for _, user := range users {
			user.CreateAt = time.Now()
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)
		}

		Convey("Find 查询所有记录", func() {
			q := &query.BoolQuery{} // 空布尔查询匹配所有
			results, err := repo.Find(ctx, q)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 4)
		})

		Convey("FindOne 查询单条记录", func() {
			q := &query.TermQuery{Field: "name", Value: "Alice"}
			result, err := repo.FindOne(ctx, q)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(result.Name, ShouldEqual, "Alice")
			So(result.Age, ShouldEqual, 25)
		})

		Convey("Count 统计记录数量", func() {
			q := &query.TermQuery{Field: "active", Value: true}
			count, err := repo.Count(ctx, q)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 3) // Alice, Charlie, Diana
		})

		Convey("Exists 检查记录是否存在", func() {
			q := &query.TermQuery{Field: "email", Value: "bob@example.com"}
			exists, err := repo.Exists(ctx, q)
			So(err, ShouldBeNil)
			So(exists, ShouldBeTrue)

			q2 := &query.TermQuery{Field: "email", Value: "nonexistent@example.com"}
			exists2, err := repo.Exists(ctx, q2)
			So(err, ShouldBeNil)
			So(exists2, ShouldBeFalse)
		})

		Convey("Find 带查询选项", func() {
			q := &query.BoolQuery{} // 空布尔查询匹配所有
			opts := []database.QueryOption{
				func(options *database.QueryOptions) {
					options.Limit = 2
					options.OrderBy = "age"
					options.OrderDesc = false
				},
			}
			results, err := repo.Find(ctx, q, opts...)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)
			So(results[0].Age, ShouldBeLessThanOrEqualTo, results[1].Age)
		})
	})
}

func TestRepositoryBatch(t *testing.T) {
	Convey("测试 Repository 批量操作", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[User](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "User")

		// 迁移表结构
		err = repo.Migrate(ctx)
		So(err, ShouldBeNil)

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "User")
		})

		Convey("BatchCreate 批量创建", func() {
			users := []*User{
				{ID: 10, Name: "User1", Email: "user1@example.com", Age: 20, Active: true, Score: 80.0, CreateAt: time.Now()},
				{ID: 11, Name: "User2", Email: "user2@example.com", Age: 21, Active: false, Score: 81.0, CreateAt: time.Now()},
				{ID: 12, Name: "User3", Email: "user3@example.com", Age: 22, Active: true, Score: 82.0, CreateAt: time.Now()},
			}

			err := repo.BatchCreate(ctx, users)
			So(err, ShouldBeNil)

			// 验证创建结果
			for _, user := range users {
				retrievedUser, err := repo.Get(ctx, user.ID)
				So(err, ShouldBeNil)
				So(retrievedUser.Name, ShouldEqual, user.Name)
			}
		})

		Convey("BatchUpdate 批量更新", func() {
			// 先创建一些用户
			users := []*User{
				{ID: 20, Name: "Original1", Email: "orig1@example.com", Age: 25, Active: true, Score: 85.0, CreateAt: time.Now()},
				{ID: 21, Name: "Original2", Email: "orig2@example.com", Age: 26, Active: false, Score: 86.0, CreateAt: time.Now()},
			}

			err := repo.BatchCreate(ctx, users)
			So(err, ShouldBeNil)

			// 更新用户信息
			users[0].Name = "Updated1"
			users[0].Age = 30
			users[1].Name = "Updated2"
			users[1].Age = 31

			err = repo.BatchUpdate(ctx, users)
			So(err, ShouldBeNil)

			// 验证更新结果
			updatedUser1, err := repo.Get(ctx, 20)
			So(err, ShouldBeNil)
			So(updatedUser1.Name, ShouldEqual, "Updated1")
			So(updatedUser1.Age, ShouldEqual, 30)

			updatedUser2, err := repo.Get(ctx, 21)
			So(err, ShouldBeNil)
			So(updatedUser2.Name, ShouldEqual, "Updated2")
			So(updatedUser2.Age, ShouldEqual, 31)
		})

		Convey("BatchDelete 批量删除", func() {
			// 先创建一些用户
			users := []*User{
				{ID: 30, Name: "ToDelete1", Email: "del1@example.com", Age: 25, Active: true, Score: 85.0, CreateAt: time.Now()},
				{ID: 31, Name: "ToDelete2", Email: "del2@example.com", Age: 26, Active: false, Score: 86.0, CreateAt: time.Now()},
				{ID: 32, Name: "ToDelete3", Email: "del3@example.com", Age: 27, Active: true, Score: 87.0, CreateAt: time.Now()},
			}

			err := repo.BatchCreate(ctx, users)
			So(err, ShouldBeNil)

			// 批量删除
			ids := []any{30, 31, 32}
			err = repo.BatchDelete(ctx, ids)
			So(err, ShouldBeNil)

			// 验证删除结果
			for _, id := range ids {
				_, err := repo.Get(ctx, id)
				So(err, ShouldEqual, database.ErrRecordNotFound)
			}
		})
	})
}

func TestRepositoryWithCreateOptions(t *testing.T) {
	Convey("测试 Repository 创建选项", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[User](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "User")

		// 迁移表结构
		err = repo.Migrate(ctx)
		So(err, ShouldBeNil)

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "User")
		})

		Convey("使用 IgnoreConflict 选项", func() {
			user := &User{
				ID:       5000,
				Name:     "Test User",
				Email:    "test@example.com",
				Age:      25,
				Active:   true,
				Score:    88.5,
				CreateAt: time.Now(),
			}

			// 第一次创建
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)

			// 使用 IgnoreConflict 选项再次创建相同 ID 的用户
			err = repo.Create(ctx, user, database.WithIgnoreConflict())
			So(err, ShouldBeNil) // 应该成功，忽略冲突
		})

		Convey("使用 UpdateOnConflict 选项", func() {
			user := &User{
				ID:       101,
				Name:     "Original Name",
				Email:    "original@example.com",
				Age:      25,
				Active:   true,
				Score:    88.5,
				CreateAt: time.Now(),
			}

			// 第一次创建
			err := repo.Create(ctx, user)
			So(err, ShouldBeNil)

			// 修改信息并使用 UpdateOnConflict 选项
			user.Name = "Updated Name"
			user.Age = 30
			err = repo.Create(ctx, user, database.WithUpdateOnConflict())
			So(err, ShouldBeNil)

			// 验证更新结果
			retrievedUser, err := repo.Get(ctx, 101)
			So(err, ShouldBeNil)
			So(retrievedUser.Name, ShouldEqual, "Updated Name")
			So(retrievedUser.Age, ShouldEqual, 30)
		})
	})
}

func TestRepositoryCompositeKey(t *testing.T) {
	Convey("测试复合主键 Repository", t, func() {
		db, err := database.NewSQLWithOptions(testMySQLOptions)
		So(err, ShouldBeNil)
		defer db.Close()

		repo, err := NewRepository[UserProfile](db)
		So(err, ShouldBeNil)

		ctx := context.Background()

		// 清理数据 - 测试开始前删除表
		_ = db.DropTable(ctx, "UserProfile")

		// 迁移表结构
		err = repo.Migrate(ctx)
		So(err, ShouldBeNil)

		// 清理函数
		Reset(func() {
			_ = db.DropTable(ctx, "UserProfile")
		})

		Convey("复合主键 CRUD 操作", func() {
			profile := &UserProfile{
				UserID:   1,
				Platform: "github",
				Username: "testuser",
				Avatar:   "avatar.png",
			}

			// 创建
			err := repo.Create(ctx, profile)
			So(err, ShouldBeNil)

			// 获取（使用复合主键）
			compositeKey := map[string]any{
				"user_id":  1,
				"platform": "github",
			}
			retrievedProfile, err := repo.Get(ctx, compositeKey)
			So(err, ShouldBeNil)
			So(retrievedProfile.Username, ShouldEqual, "testuser")
			So(retrievedProfile.Avatar, ShouldEqual, "avatar.png")

			// 更新
			profile.Username = "updateduser"
			profile.Avatar = "new_avatar.png"
			err = repo.Update(ctx, profile)
			So(err, ShouldBeNil)

			// 验证更新
			updatedProfile, err := repo.Get(ctx, compositeKey)
			So(err, ShouldBeNil)
			So(updatedProfile.Username, ShouldEqual, "updateduser")
			So(updatedProfile.Avatar, ShouldEqual, "new_avatar.png")

			// 删除
			err = repo.Delete(ctx, compositeKey)
			So(err, ShouldBeNil)

			// 验证删除
			_, err = repo.Get(ctx, compositeKey)
			So(err, ShouldEqual, database.ErrRecordNotFound)
		})
	})
}