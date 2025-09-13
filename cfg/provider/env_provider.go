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
}

type EnvProviderOptions struct {
	EnvFiles []string
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
	}, nil
}

func (p *EnvProvider) Load() ([]byte, error) {
	envVars := make(map[string]string)

	// 首先加载系统环境变量（优先级最低）
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
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

		// 不处理引号和转义，保持原始值，交给 decoder 处理
		envVars[key] = value
	}

	return scanner.Err()
}

func (p *EnvProvider) Save(data []byte) error {
	return errors.New("env provider does not support save operation")
}

func (p *EnvProvider) OnChange(fn func(data []byte) error) {
	// 不支持变更监听，直接返回
}

func (p *EnvProvider) Watch() error {
	return errors.New("env provider does not support watch operation")
}

func (p *EnvProvider) Close() error {
	// 无需释放资源
	return nil
}
