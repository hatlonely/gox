package cfg

import (
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log/logger"
)

// Config 配置接口，定义了配置对象的基本操作
// 提供配置数据的统一访问入口和变更监听功能
type Config interface {
	// Sub 获取子配置对象
	// 当key为空字符串时，返回自身（与Storage.Sub("")的行为一致）
	Sub(key string) Config

	// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
	ConvertTo(object any) error

	// SetLogger 设置日志记录器（只有根配置才能设置）
	SetLogger(logger logger.Logger)

	// OnChange 监听配置变更
	OnChange(fn func(storage.Storage) error)

	// OnKeyChange 监听指定键的配置变更
	OnKeyChange(key string, fn func(storage.Storage) error)

	// Watch 启动配置变更监听
	// 只有调用此方法后，OnChange 和 OnKeyChange 注册的回调函数才会被触发
	// 对于不支持监听的 Provider，此方法静默处理不返回错误
	// 为了防止在 NewConfig 和 Watch 之间丢失配置变更，会主动检查一次配置
	Watch() error

	// Close 关闭配置对象，释放相关资源
	// 只有根配置对象才能执行关闭操作，子配置对象会将关闭请求转发到根配置
	// 多次调用只会执行一次，后续调用直接返回第一次调用的结果
	Close() error
}
