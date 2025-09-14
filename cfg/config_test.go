package cfg

import (
	"testing"

	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/refx"
)

// TestConfigInterfaceImplementation 验证 SingleConfig 实现了 Config 接口
func TestConfigInterfaceImplementation(t *testing.T) {
	// 创建一个 SingleConfig 实例
	options := &Options{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "EnvProvider",
			Options:   &provider.EnvProviderOptions{},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "EnvDecoder",
			Options:   nil,
		},
	}

	singleConfig, err := NewSingleConfigWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create SingleConfig: %v", err)
	}
	defer singleConfig.Close()

	// 验证 SingleConfig 可以赋值给 Config 接口
	var config Config = singleConfig

	// 验证接口方法可以正常调用
	subConfig := config.Sub("test")
	if subConfig == nil {
		t.Error("Sub method should return a valid Config")
	}

	// 验证其他接口方法
	err = config.ConvertTo(&map[string]any{})
	if err != nil {
		t.Logf("ConvertTo failed (expected for env provider): %v", err)
	}

	config.OnChange(func(s storage.Storage) error {
		return nil
	})

	config.OnKeyChange("test", func(s storage.Storage) error {
		return nil
	})

	err = config.Watch()
	if err != nil {
		t.Logf("Watch failed (expected for env provider): %v", err)
	}

	t.Log("SingleConfig successfully implements Config interface")
}
