package provider

import (
	"testing"
	"time"

	"github.com/hatlonely/gox/rdb/database"
	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
)

func TestRdbProvider(t *testing.T) {
	Convey("测试 RdbProvider", t, func() {
		// 使用 MySQL 测试数据库
		options := &RdbProviderOptions{
			ConfigID: "test-config",
			Database: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/rdb/database",
				Type:      "SQL",
				Options: &database.SQLOptions{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     "3306",
					Database: "testdb",
					Username: "testuser",
					Password: "testpass",
					Charset:  "utf8mb4",
				},
			},
			PollInterval: 100 * time.Millisecond,
		}

		provider, err := NewRdbProviderWithOptions(options)
		So(err, ShouldBeNil)
		So(provider, ShouldNotBeNil)

		defer provider.Close()

		Convey("测试保存和读取配置", func() {
			testData := []byte(`{"key": "value"}`)

			// 保存配置
			err := provider.Save(testData)
			So(err, ShouldBeNil)

			// 读取配置
			data, err := provider.Load()
			So(err, ShouldBeNil)
			So(string(data), ShouldEqual, string(testData))
		})

		Convey("测试更新配置", func() {
			// 第一次保存
			testData1 := []byte(`{"key": "value1"}`)
			err := provider.Save(testData1)
			So(err, ShouldBeNil)

			// 第二次保存（更新）
			testData2 := []byte(`{"key": "value2"}`)
			err = provider.Save(testData2)
			So(err, ShouldBeNil)

			// 读取配置
			data, err := provider.Load()
			So(err, ShouldBeNil)
			So(string(data), ShouldEqual, string(testData2))
		})

		Convey("测试配置变更监听", func() {
			// 注册变更回调
			var receivedData []byte
			callbackCalled := make(chan bool, 1)
			
			provider.OnChange(func(data []byte) error {
				receivedData = data
				callbackCalled <- true
				return nil
			})

			// 启动监听
			err := provider.Watch()
			So(err, ShouldBeNil)

			// 保存初始配置
			initialData := []byte(`{"initial": "data"}`)
			err = provider.Save(initialData)
			So(err, ShouldBeNil)

			// 更新配置
			updatedData := []byte(`{"updated": "data"}`)
			err = provider.Save(updatedData)
			So(err, ShouldBeNil)

			// 等待回调被调用
			select {
			case <-callbackCalled:
				So(string(receivedData), ShouldEqual, string(updatedData))
			case <-time.After(500 * time.Millisecond):
				So(false, ShouldBeTrue) // 超时失败
			}
		})

		Convey("测试读取不存在的配置", func() {
			// 创建一个新的 provider 使用不存在的配置 ID
			newOptions := &RdbProviderOptions{
				ConfigID: "non-existent-config",
				Database: options.Database,
			}

			newProvider, err := NewRdbProviderWithOptions(newOptions)
			So(err, ShouldBeNil)
			defer newProvider.Close()

			// 尝试读取不存在的配置
			_, err = newProvider.Load()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestRdbProviderValidation(t *testing.T) {
	Convey("测试 RdbProvider 参数验证", t, func() {
		Convey("空 options", func() {
			provider, err := NewRdbProviderWithOptions(nil)
			So(err, ShouldNotBeNil)
			So(provider, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "rdb provider options is required")
		})

		Convey("空 ConfigID", func() {
			options := &RdbProviderOptions{
				Database: &ref.TypeOptions{
					Type: "SQL",
				},
			}
			provider, err := NewRdbProviderWithOptions(options)
			So(err, ShouldNotBeNil)
			So(provider, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "config ID is required")
		})

		Convey("空 Database 配置", func() {
			options := &RdbProviderOptions{
				ConfigID: "test",
			}
			provider, err := NewRdbProviderWithOptions(options)
			So(err, ShouldNotBeNil)
			So(provider, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "database config is required")
		})
	})
}