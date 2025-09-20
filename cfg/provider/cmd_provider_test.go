package provider

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewCmdProviderWithOptions(t *testing.T) {
	Convey("测试NewCmdProviderWithOptions函数", t, func() {
		Convey("使用nil选项", func() {
			provider, err := NewCmdProviderWithOptions(nil)
			So(err, ShouldBeNil)
			So(provider, ShouldNotBeNil)
		})

		Convey("使用空选项", func() {
			provider, err := NewCmdProviderWithOptions(&CmdProviderOptions{})
			So(err, ShouldBeNil)
			So(provider, ShouldNotBeNil)
		})

		Convey("使用带前缀的选项", func() {
			provider, err := NewCmdProviderWithOptions(&CmdProviderOptions{
				Prefix: "app-",
			})
			So(err, ShouldBeNil)
			So(provider, ShouldNotBeNil)
		})
	})
}

func TestCmdProvider_Load(t *testing.T) {
	Convey("测试CmdProvider的Load功能", t, func() {
		testLoad := func(name, prefix string, testArgs []string, want map[string]string) {
			Convey(name, func() {
				provider := &CmdProvider{
					prefix:   prefix,
					testArgs: testArgs,
				}

				data, err := provider.Load()
				So(err, ShouldBeNil)

				// 解析返回的 .env 格式数据
				got := parseCmdData(string(data))

				// 检查期望的键值对
				for key, expectedValue := range want {
					So(got, ShouldContainKey, key)
					So(got[key], ShouldEqual, expectedValue)
				}

				// 检查是否有多余的键
				So(len(got), ShouldEqual, len(want))
			})
		}

		testLoad("无参数", "", []string{}, map[string]string{})

		testLoad("简单的key=value格式", "", []string{"--host=localhost", "--port=3306"}, 
			map[string]string{"host": "localhost", "port": "3306"})

		testLoad("key value格式", "", []string{"--host", "localhost", "--port", "3306"}, 
			map[string]string{"host": "localhost", "port": "3306"})

		testLoad("布尔标志", "", []string{"--debug", "--verbose"}, 
			map[string]string{"debug": "true", "verbose": "true"})

		testLoad("混合格式", "", []string{"--host=localhost", "--port", "3306", "--debug", "--name", "test app"}, 
			map[string]string{"host": "localhost", "port": "3306", "debug": "true", "name": "test app"})

		testLoad("带前缀过滤", "app-", []string{"--app-host=localhost", "--app-port=3306", "--debug", "--other=ignored"}, 
			map[string]string{"host": "localhost", "port": "3306"})

		testLoad("包含空格和特殊字符的值", "", []string{"--message=hello world", "--key", "secret key with spaces", "--json={\"test\":true}"}, 
			map[string]string{"message": "hello world", "key": "secret key with spaces", "json": "{\"test\":true}"})

		testLoad("复合结构键", "", []string{"--redis-password", "123456", "--database-url=postgres://localhost:5432/db", "--jwt-secret=mysecret", "--api-timeout", "30s", "--cache-redis-addr=localhost:6379"}, 
			map[string]string{"redis-password": "123456", "database-url": "postgres://localhost:5432/db", "jwt-secret": "mysecret", "api-timeout": "30s", "cache-redis-addr": "localhost:6379"})

		testLoad("嵌套复合键与前缀", "app-", []string{"--app-redis-host=localhost", "--app-redis-port", "6379", "--app-db-mysql-host=127.0.0.1", "--other-key=ignored", "--app-log-level=debug"}, 
			map[string]string{"redis-host": "localhost", "redis-port": "6379", "db-mysql-host": "127.0.0.1", "log-level": "debug"})

		testLoad("忽略非长选项", "", []string{"-h", "help", "--host=localhost", "ignored", "--port", "3306"}, 
			map[string]string{"host": "localhost", "port": "3306"})

		testLoad("空值", "", []string{"--empty=", "--host", ""}, 
			map[string]string{"empty": "", "host": ""})

		testLoad("复杂嵌套配置键", "", 
			[]string{"--server-http-port=8080", "--server-grpc-port", "9090", "--database-mysql-master-host=db1.example.com", "--database-mysql-slave-host", "db2.example.com", "--redis-cluster-node-1-addr=redis1:6379", "--redis-cluster-node-2-addr=redis2:6379", "--oauth2-google-client-id=12345", "--feature-flag-new-ui-enabled"}, 
			map[string]string{"server-http-port": "8080", "server-grpc-port": "9090", "database-mysql-master-host": "db1.example.com", "database-mysql-slave-host": "db2.example.com", "redis-cluster-node-1-addr": "redis1:6379", "redis-cluster-node-2-addr": "redis2:6379", "oauth2-google-client-id": "12345", "feature-flag-new-ui-enabled": "true"})

		testLoad("Kubernetes风格配置", "k8s-", 
			[]string{"--k8s-namespace=default", "--k8s-service-account", "myapp", "--k8s-config-map-name=app-config", "--k8s-secret-tls-cert-path=/etc/ssl/certs/tls.crt", "--other-flag=ignored", "--k8s-ingress-class-name", "nginx"}, 
			map[string]string{"namespace": "default", "service-account": "myapp", "config-map-name": "app-config", "secret-tls-cert-path": "/etc/ssl/certs/tls.crt", "ingress-class-name": "nginx"})
	})
}

func TestCmdProvider_Save(t *testing.T) {
	Convey("测试CmdProvider的Save功能", t, func() {
		provider, err := NewCmdProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("Save操作应该返回错误（不支持保存）", func() {
			err = provider.Save([]byte("test data"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "does not support save")
		})
	})
}

func TestCmdProvider_OnChange(t *testing.T) {
	Convey("测试CmdProvider的OnChange功能", t, func() {
		provider, err := NewCmdProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("OnChange应该直接返回，不做任何操作", func() {
			called := false
			provider.OnChange(func(data []byte) error {
				called = true
				return nil
			})
			// 验证回调没有被调用（因为不支持变更监听）
			So(called, ShouldBeFalse)
		})
	})
}

func TestCmdProvider_Close(t *testing.T) {
	Convey("测试CmdProvider的Close功能", t, func() {
		provider, err := NewCmdProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("Close操作应该成功", func() {
			err = provider.Close()
			So(err, ShouldBeNil)
		})
	})
}

func TestCmdProvider_EdgeCases(t *testing.T) {
	Convey("测试CmdProvider的边界情况", t, func() {
		testEdgeCase := func(name, prefix string, testArgs []string, want map[string]string) {
			Convey(name, func() {
				provider := &CmdProvider{
					prefix:   prefix,
					testArgs: testArgs,
				}

				data, err := provider.Load()
				So(err, ShouldBeNil)

				got := parseCmdData(string(data))

				// 检查期望的键值对
				for key, expectedValue := range want {
					So(got, ShouldContainKey, key)
					So(got[key], ShouldEqual, expectedValue)
				}

				// 检查是否有多余的键
				So(len(got), ShouldEqual, len(want))
			})
		}

		testEdgeCase("--后的空键", "", []string{"--"}, map[string]string{})

		testEdgeCase("带等号但值为空的键", "", []string{"--key="}, 
			map[string]string{"key": ""})

		testEdgeCase("多个等号", "", []string{"--url=http://localhost:3000/path?param=value"}, 
			map[string]string{"url": "http://localhost:3000/path?param=value"})

		testEdgeCase("前缀边界情况", "app-", []string{"--app-", "--app-host=localhost"}, 
			map[string]string{"host": "localhost"})
	})
}

// parseCmdData 解析 .env 格式数据为 map
func parseCmdData(data string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		equalIndex := strings.Index(line, "=")
		if equalIndex == -1 {
			continue
		}

		key := strings.TrimSpace(line[:equalIndex])
		value := line[equalIndex+1:]

		if key != "" {
			result[key] = value
		}
	}

	return result
}

func TestCmdProvider_Watch(t *testing.T) {
	Convey("测试CmdProvider的Watch功能", t, func() {
		provider, err := NewCmdProviderWithOptions(nil)
		So(err, ShouldBeNil)
		defer provider.Close()

		Convey("CmdProvider不支持Watch，但不应该返回错误", func() {
			err = provider.Watch()
			So(err, ShouldBeNil)
		})
	})
}
