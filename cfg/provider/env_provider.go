package provider

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type EnvProvider struct {
	envFiles []string
	prefix   string
}

type EnvProviderOptions struct {
	EnvFiles []string
	// Prefix 环境变量前缀过滤，如 "APP_" 只处理 APP_ 开头的环境变量，处理时直接移除前缀
	Prefix string
}

func NewEnvProviderWithOptions(options *EnvProviderOptions) (*EnvProvider, error) {
	if options == nil {
		options = &EnvProviderOptions{}
	}

	var envFiles []string
	for _, file := range options.EnvFiles {
		if file != "" {
			absPath, err := filepath.Abs(file)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid env file path: %s", file)
			}
			envFiles = append(envFiles, absPath)
		}
	}

	return &EnvProvider{
		envFiles: envFiles,
		prefix:   options.Prefix,
	}, nil
}

func (p *EnvProvider) Load() ([]byte, error) {
	envVars := make(map[string]string)

	// 首先加载系统环境变量（优先级最低）
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			
			// 如果设置了前缀，只处理匹配前缀的环境变量
			if p.prefix != "" {
				if !strings.HasPrefix(key, p.prefix) {
					continue
				}
				// 移除前缀
				key = key[len(p.prefix):]
				if key == "" {
					continue
				}
			}
			
			envVars[key] = value
		}
	}

	// 按顺序加载 .env 文件，后面的文件会覆盖前面的
	for _, envFile := range p.envFiles {
		if err := p.loadEnvFile(envFile, envVars); err != nil {
			// 文件不存在时不报错，继续处理其他文件
			if !os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "failed to load env file: %s", envFile)
			}
		}
	}

	// 将环境变量转换为 .env 格式，不做引号处理，交给 decoder 处理
	var lines []string
	for key, value := range envVars {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return []byte(strings.Join(lines, "\n")), nil
}

func (p *EnvProvider) loadEnvFile(filename string, envVars map[string]string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// 解析键值对
		equalIndex := strings.Index(line, "=")
		if equalIndex == -1 {
			continue // 跳过无效行，不报错
		}

		key := strings.TrimSpace(line[:equalIndex])
		value := line[equalIndex+1:]

		if key == "" {
			continue // 跳过空键名
		}

		// 应用前缀过滤逻辑
		key = p.filterKeyWithPrefix(key)
		if key == "" {
			continue // 前缀不匹配，跳过
		}

		// 不处理引号和转义，保持原始值，交给 decoder 处理
		envVars[key] = value
	}

	return scanner.Err()
}

// filterKeyWithPrefix 根据前缀过滤和处理键名
// 返回空字符串表示应该跳过这个键
func (p *EnvProvider) filterKeyWithPrefix(key string) string {
	// 如果没有设置前缀，直接返回原键名
	if p.prefix == "" {
		return key
	}

	// 如果键名不以前缀开头，返回空字符串表示跳过
	if !strings.HasPrefix(key, p.prefix) {
		return ""
	}

	// 移除前缀
	filteredKey := key[len(p.prefix):]
	if filteredKey == "" {
		return "" // 移除前缀后为空，跳过
	}

	return filteredKey
}

func (p *EnvProvider) Save(data []byte) error {
	return errors.New("env provider does not support save operation")
}

func (p *EnvProvider) OnChange(fn func(data []byte) error) {
	// 不支持变更监听，直接返回
}

func (p *EnvProvider) Watch() error {
	// env provider 不支持变更监听，静默处理
	return nil
}

func (p *EnvProvider) Close() error {
	// 无需释放资源
	return nil
}
