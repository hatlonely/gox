package cfg

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/refx"
)

// NewConfig 简化构造方法，从文件读取基础配置，同时支持环境变量和命令行覆盖
// 
// 配置优先级（从低到高）：文件 < 环境变量 < 命令行
// 
// 支持的文件格式：
//   - .json/.json5 -> JsonDecoder
//   - .yaml/.yml -> YamlDecoder  
//   - .toml -> TomlDecoder
//   - .ini -> IniDecoder
//   - .env -> EnvDecoder
//
// 环境变量：读取所有系统环境变量
// 命令行：处理所有 --key=value 或 --key value 格式的参数
//
// 使用示例：
//   cfg, err := NewConfig("config.yaml")
//   if err != nil {
//       return err
//   }
//   defer cfg.Close()
//
//   // 环境变量 DATABASE_HOST=localhost 会覆盖文件中的 database.host
//   // 命令行 --database-port=3306 会覆盖环境变量和文件中的 database.port
func NewConfig(filename string) (Config, error) {
	return NewConfigWithPrefix(filename, "", "")
}

// NewConfigWithPrefix 简化构造方法，支持指定环境变量和命令行参数前缀
// 
// 参数：
//   - filename: 配置文件路径
//   - envPrefix: 环境变量前缀，如 "APP_"，会过滤 APP_ 开头的环境变量并移除前缀
//   - cmdPrefix: 命令行参数前缀，如 "app-"，会处理 --app-* 参数并移除前缀
//
// 使用示例：
//   cfg, err := NewConfigWithPrefix("config.yaml", "APP_", "app-")
//   // 只处理 APP_* 环境变量和 --app-* 命令行参数
func NewConfigWithPrefix(filename, envPrefix, cmdPrefix string) (Config, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// 创建配置源选项
	sources := make([]*ConfigSourceOptions, 0, 3)

	// 1. 文件配置源（优先级最低）
	fileSourceOptions, err := createFileSourceOptions(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file source options: %w", err)
	}
	sources = append(sources, fileSourceOptions)

	// 2. 环境变量配置源（优先级中等）
	envSourceOptions := createEnvSourceOptions(envPrefix)
	sources = append(sources, envSourceOptions)

	// 3. 命令行配置源（优先级最高）
	cmdSourceOptions := createCmdSourceOptions(cmdPrefix)
	sources = append(sources, cmdSourceOptions)

	// 创建 MultiConfig
	options := &MultiConfigOptions{
		Sources: sources,
	}

	return NewMultiConfigWithOptions(options)
}

// createFileSourceOptions 创建文件配置源选项
func createFileSourceOptions(filename string) (*ConfigSourceOptions, error) {
	// 根据文件扩展名确定解码器类型
	ext := strings.ToLower(filepath.Ext(filename))
	var decoderType string
	var decoderOptions any

	switch ext {
	case ".json", ".json5":
		decoderType = "JsonDecoder"
		decoderOptions = &decoder.JsonDecoderOptions{UseJSON5: ext == ".json5"}
	case ".yaml", ".yml":
		decoderType = "YamlDecoder"
		decoderOptions = &decoder.YamlDecoderOptions{Indent: 2}
	case ".toml":
		decoderType = "TomlDecoder"
		decoderOptions = &decoder.TomlDecoderOptions{Indent: "  "}
	case ".ini":
		decoderType = "IniDecoder"
		decoderOptions = &decoder.IniDecoderOptions{
			AllowEmptyValues: true,
			AllowBoolKeys:    true,
			AllowShadows:     true,
		}
	case ".env":
		decoderType = "EnvDecoder"
		decoderOptions = nil
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	return &ConfigSourceOptions{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "FileProvider",
			Options: &provider.FileProviderOptions{
				FilePath: filename,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      decoderType,
			Options:   decoderOptions,
		},
	}, nil
}

// createEnvSourceOptions 创建环境变量配置源选项
func createEnvSourceOptions(prefix string) *ConfigSourceOptions {
	return &ConfigSourceOptions{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "EnvProvider",
			Options: &provider.EnvProviderOptions{
				Prefix: prefix,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "EnvDecoder",
			Options:   nil,
		},
	}
}

// createCmdSourceOptions 创建命令行配置源选项
func createCmdSourceOptions(prefix string) *ConfigSourceOptions {
	return &ConfigSourceOptions{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "CmdProvider",
			Options: &provider.CmdProviderOptions{
				Prefix: prefix,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "CmdDecoder",
			Options:   nil,
		},
	}
}

