package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type CmdProvider struct {
	prefix string
	// testArgs 用于测试时模拟命令行参数
	testArgs []string
}

type CmdProviderOptions struct {
	// Prefix 参数前缀过滤，如 "app-" 只处理 --app-* 参数，处理时直接移除前缀
	Prefix string
}

func NewCmdProviderWithOptions(options *CmdProviderOptions) (*CmdProvider, error) {
	if options == nil {
		options = &CmdProviderOptions{}
	}

	return &CmdProvider{
		prefix: options.Prefix,
	}, nil
}

func (p *CmdProvider) Load() ([]byte, error) {
	var args []string
	if p.testArgs != nil {
		args = p.testArgs
	} else {
		args = os.Args[1:]
	}
	cmdVars := make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// 只处理以 -- 开头的长选项
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		// 移除 -- 前缀
		key := arg[2:]
		if key == "" {
			continue
		}

		// 检查前缀过滤
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

		var value string
		var hasValue bool

		// 检查是否包含 = 分隔符
		if equalIndex := strings.Index(key, "="); equalIndex != -1 {
			// 格式：--key=value
			value = key[equalIndex+1:]
			key = key[:equalIndex]
			hasValue = true
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			// 格式：--key value (下一个参数不是选项)
			i++ // 跳过下一个参数
			value = args[i]
			hasValue = true
		}

		// 如果没有值，作为布尔标志处理
		if !hasValue {
			value = "true"
		}

		cmdVars[key] = value
	}

	// 将命令行参数转换为 .env 格式
	var lines []string
	for key, value := range cmdVars {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return []byte(strings.Join(lines, "\n")), nil
}

func (p *CmdProvider) Save(data []byte) error {
	return errors.New("cmd provider does not support save operation")
}

func (p *CmdProvider) OnChange(fn func(data []byte) error) {
	// 命令行参数是静态的，不支持变更监听
}

func (p *CmdProvider) Watch() error {
	// cmd provider 不支持变更监听，静默处理
	return nil
}

func (p *CmdProvider) Close() error {
	// 无需释放资源
	return nil
}
