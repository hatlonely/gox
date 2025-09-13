package provider

import "github.com/hatlonely/gox/refx"

func init() {
	refx.MustRegisterT[FileProvider](NewFileProviderWithOptions)
	refx.MustRegisterT[GormProvider](NewGormProviderWithOptions)
	refx.MustRegisterT[EnvProvider](NewEnvProviderWithOptions)
	refx.MustRegisterT[CmdProvider](NewCmdProviderWithOptions)

	refx.MustRegisterT[*FileProvider](NewFileProviderWithOptions)
	refx.MustRegisterT[*GormProvider](NewGormProviderWithOptions)
	refx.MustRegisterT[*EnvProvider](NewEnvProviderWithOptions)
	refx.MustRegisterT[*CmdProvider](NewCmdProviderWithOptions)
}

// Provider 配置数据提供者接口
// 负责读取配置数据和监听配置变更
type Provider interface {
	// Load 读取配置数据
	Load() (data []byte, err error)
	// Save 保存配置数据
	Save(data []byte) error
	// OnChange 注册配置数据变更回调函数
	// 此方法仅仅添加回调函数，不启动监听
	OnChange(fn func(data []byte) error)
	// Watch 启动配置变更监听
	// 只有调用此方法后，OnChange 注册的回调函数才会被触发
	Watch() error
	// Close 关闭提供者，释放资源
	Close() error
}
