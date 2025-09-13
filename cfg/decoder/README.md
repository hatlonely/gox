# Decoder

配置数据编解码器包，提供多种格式的配置文件解析和生成功能。

## 接口设计

所有解码器都实现了统一的 `Decoder` 接口：

```go
type Decoder interface {
    // Decode 将原始数据解码为存储对象
    Decode(data []byte) (storage.Storage, error)
    // Encode 将存储对象编码为原始数据
    Encode(storage.Storage) ([]byte, error)
}
```

## 构造方法设计

所有解码器都采用统一的构造方法模式：

- `NewXXXDecoder()`: 使用默认配置创建解码器
- `NewXXXDecoderWithOptions(options *XXXDecoderOptions)`: 使用自定义配置创建解码器
- 当 `options` 参数为 `nil` 时，自动使用默认配置

## 支持的格式

### JSON 解码器 (`JsonDecoder`)
- 支持标准 JSON 格式
- 支持 JSON5 格式（注释、尾随逗号）
- 自动移除单行和多行注释
- 默认启用 JSON5 支持

```go
decoder := NewJsonDecoder()  // 默认启用 JSON5
decoder := NewJsonDecoderWithOptions(&JsonDecoderOptions{
    UseJSON5: false,  // 仅标准 JSON
})
```

### YAML 解码器 (`YamlDecoder`)
- 支持标准 YAML 格式
- 内置注释支持
- 可配置缩进空格数

```go
decoder := NewYamlDecoder()  // 默认 2 空格缩进
decoder := NewYamlDecoderWithOptions(&YamlDecoderOptions{
    Indent: 4,  // 4 空格缩进
})
```

### TOML 解码器 (`TomlDecoder`)
- 支持标准 TOML 格式
- 内置注释支持
- 可配置输出缩进

```go
decoder := NewTomlDecoder()  // 默认 2 空格缩进
decoder := NewTomlDecoderWithOptions(&TomlDecoderOptions{
    Indent: "\t",  // Tab 缩进
})
```

### INI 解码器 (`IniDecoder`)
- 支持标准 INI 格式
- 支持注释和分组
- 支持空值和布尔键
- 支持重复键（生成数组）
- 自动类型转换（布尔值、数字、数组）

```go
decoder := NewIniDecoder()  // 默认所有选项开启
decoder := NewIniDecoderWithOptions(&IniDecoderOptions{
    AllowEmptyValues: true,
    AllowBoolKeys:    true,
    AllowShadows:     false,
})
```

### 环境变量解码器 (`EnvDecoder`)
- 支持 .env 文件格式
- 使用 `FlatStorage` 进行智能字段映射
- 固定配置：键分隔符 `_`，数组格式 `_%d`，支持注释和空行
- 自动类型转换（布尔值、数字）
- 支持引号包围的字符串

```go
decoder := NewEnvDecoder()  // 使用固定的默认配置
decoder := NewEnvDecoderWithOptions(nil)  // 兼容性方法，忽略配置
```

### 命令行参数解码器 (`CmdDecoder`)
- 支持命令行参数格式
- 使用 `FlatStorage` 进行智能字段映射
- 支持 kebab-case 键名（`server-http-port`）
- 自动类型转换（布尔值、数字）
- 支持引号包围的字符串和转义字符
- 可配置键分隔符和数组格式

```go
decoder := NewCmdDecoder()  // 默认使用 "-" 分隔符
decoder := NewCmdDecoderWithOptions(&CmdDecoderOptions{
    Separator:     "_",
    ArrayFormat:   "_%d", 
    AllowComments: true,
    AllowEmpty:    true,
})
```

## 使用示例

### 基本使用

```go
// 使用默认配置
jsonDecoder := decoder.NewJsonDecoder()
yamlDecoder := decoder.NewYamlDecoder()
tomlDecoder := decoder.NewTomlDecoder()
iniDecoder := decoder.NewIniDecoder()
envDecoder := decoder.NewEnvDecoder()
cmdDecoder := decoder.NewCmdDecoder()

// 解码数据
storage, err := jsonDecoder.Decode([]byte(`{"key": "value"}`))
storage, err := yamlDecoder.Decode([]byte(`key: value`))
storage, err := tomlDecoder.Decode([]byte(`key = "value"`))
storage, err := iniDecoder.Decode([]byte(`key=value`))
storage, err := envDecoder.Decode([]byte(`KEY=value`))
storage, err := cmdDecoder.Decode([]byte(`server-port=8080`))
```

### 自定义配置

```go
// 使用自定义配置
jsonDecoder := decoder.NewJsonDecoderWithOptions(&decoder.JsonDecoderOptions{
    UseJSON5: false,
})

yamlDecoder := decoder.NewYamlDecoderWithOptions(&decoder.YamlDecoderOptions{
    Indent: 4,
})

// EnvDecoder 已简化，不再需要配置选项
envDecoder := decoder.NewEnvDecoder()

cmdDecoder := decoder.NewCmdDecoderWithOptions(&decoder.CmdDecoderOptions{
    Separator:     "_",
    ArrayFormat:   "_%d",
    AllowComments: false,
    AllowEmpty:    false,
})

// 传递 nil 使用默认配置
iniDecoder := decoder.NewIniDecoderWithOptions(nil)
```

## 特性

- 统一的接口设计，便于切换不同格式
- 自动类型转换和智能解析
- 完善的错误处理和验证
- 支持嵌套结构和数组
- 保持格式化输出的可读性