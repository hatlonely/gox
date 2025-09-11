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

## 支持的格式

### JSON 解码器 (`JsonDecoder`)
- 支持标准 JSON 格式
- 支持 JSON5 格式（注释、尾随逗号）
- 自动移除单行和多行注释
- 默认启用 JSON5 支持

```go
decoder := NewJsonDecoder()  // 默认启用 JSON5
decoder := NewJsonDecoderWithOptions(false)  // 仅标准 JSON
```

### YAML 解码器 (`YamlDecoder`)
- 支持标准 YAML 格式
- 内置注释支持
- 可配置缩进空格数

```go
decoder := NewYamlDecoder()  // 默认 2 空格缩进
decoder := NewYamlDecoderWithIndent(4)  // 4 空格缩进
```

### TOML 解码器 (`TomlDecoder`)
- 支持标准 TOML 格式
- 内置注释支持
- 可配置输出缩进

```go
decoder := NewTomlDecoder()  // 默认 2 空格缩进
decoder := NewTomlDecoderWithIndent("\t")  // Tab 缩进
```

### INI 解码器 (`IniDecoder`)
- 支持标准 INI 格式
- 支持注释和分组
- 支持空值和布尔键
- 支持重复键（生成数组）
- 自动类型转换（布尔值、数字、数组）

```go
decoder := NewIniDecoder()  // 默认所有选项开启
decoder := NewIniDecoderWithOptions(
    true,   // AllowEmptyValues
    true,   // AllowBoolKeys
    false,  // AllowShadows
)
```

### 环境变量解码器 (`EnvDecoder`)
- 支持 .env 文件格式
- 使用 `FlatStorage` 进行智能字段映射
- 支持注释和空行
- 自动类型转换（布尔值、数字）
- 支持引号包围的字符串
- 可配置键分隔符和数组格式

```go
decoder := NewEnvDecoder()  // 默认使用 "_" 分隔符
decoder := NewEnvDecoderWithOptions(".", "[%d]")  // 自定义分隔符
```

## 使用示例

```go
// JSON
jsonDecoder := decoder.NewJsonDecoder()
storage, err := jsonDecoder.Decode([]byte(`{"key": "value"}`))

// YAML
yamlDecoder := decoder.NewYamlDecoder()
storage, err := yamlDecoder.Decode([]byte(`key: value`))

// TOML
tomlDecoder := decoder.NewTomlDecoder()
storage, err := tomlDecoder.Decode([]byte(`key = "value"`))

// INI
iniDecoder := decoder.NewIniDecoder()
storage, err := iniDecoder.Decode([]byte(`key=value`))

// ENV
envDecoder := decoder.NewEnvDecoder()
storage, err := envDecoder.Decode([]byte(`KEY=value`))
```

## 特性

- 统一的接口设计，便于切换不同格式
- 自动类型转换和智能解析
- 完善的错误处理和验证
- 支持嵌套结构和数组
- 保持格式化输出的可读性