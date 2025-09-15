package cfg

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// FieldInfo 字段信息结构
type FieldInfo struct {
	Path         string   // 字段路径，如 "database.host"
	Type         string   // 字段类型描述
	Help         string   // 帮助信息
	EnvName      string   // 环境变量名
	CmdName      string   // 命令行参数名
	DefaultValue string   // 默认值
	Required     bool     // 是否必填
	Examples     []string // 示例值
}

// GenerateHelp 生成配置帮助信息
// config: 配置结构体实例
// envPrefix: 环境变量前缀，如 "APP_"
// cmdPrefix: 命令行参数前缀，如 "app-"
func GenerateHelp(config interface{}, envPrefix, cmdPrefix string) string {
	fields := extractFieldInfo(config, "", envPrefix, cmdPrefix)

	if len(fields) == 0 {
		return "未找到配置字段信息"
	}

	var sb strings.Builder
	sb.WriteString("配置参数说明：\n\n")

	// 按路径排序
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})

	// 直接显示所有字段信息，不分组
	for _, field := range fields {
		sb.WriteString(formatFieldHelp(field))
		sb.WriteString("\n")
	}

	// 添加类型说明
	sb.WriteString(generateTypeHelp())

	return sb.String()
}

// extractFieldInfo 提取字段信息，只在叶子节点生成 FieldInfo
func extractFieldInfo(obj interface{}, prefix, envPrefix, cmdPrefix string) []FieldInfo {
	var fields []FieldInfo

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return fields
	}

	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// 跳过无法访问的字段
		if !fieldValue.CanInterface() {
			continue
		}

		// 获取字段配置名
		fieldName := getFieldConfigName(field)
		if fieldName == "-" {
			continue // 跳过被忽略的字段
		}

		// 构建完整路径
		fullPath := fieldName
		if prefix != "" {
			fullPath = prefix + "." + fieldName
		}

		// 获取帮助信息
		helpText := field.Tag.Get("help")
		if helpText == "" {
			helpText = generateDefaultHelp(field)
		}

		// 递归处理不同类型的字段
		fields = append(fields, extractFieldsRecursive(fieldValue, field, fullPath, helpText, envPrefix, cmdPrefix)...)
	}

	return fields
}

// extractFieldsRecursive 递归提取字段信息
func extractFieldsRecursive(fieldValue reflect.Value, field reflect.StructField, fullPath, helpText, envPrefix, cmdPrefix string) []FieldInfo {
	var fields []FieldInfo

	switch fieldValue.Kind() {
	case reflect.Struct:
		// 嵌套结构体，递归处理
		if isTimeType(fieldValue.Type()) {
			// time.Time 和 time.Duration 作为基本类型处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix))
		} else {
			// 递归处理嵌套结构体
			nestedFields := extractFieldInfo(fieldValue.Interface(), fullPath, envPrefix, cmdPrefix)
			fields = append(fields, nestedFields...)
		}

	case reflect.Slice:
		// 切片类型，递归处理元素字段
		if elemType := fieldValue.Type().Elem(); elemType.Kind() == reflect.Struct {
			// 创建元素实例来分析字段
			elemInstance := reflect.New(elemType).Interface()
			// 生成元素字段信息，使用占位符格式
			elemFields := extractFieldInfo(elemInstance, fullPath+"[N]", envPrefix, cmdPrefix)
			fields = append(fields, elemFields...)
		} else {
			// 基本类型数组，直接作为叶子节点处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix))
		}

	case reflect.Map:
		// Map类型，处理值类型
		valueType := fieldValue.Type().Elem()

		// 如果值类型是结构体，递归处理使用占位符 {KEY}
		if valueType.Kind() == reflect.Struct {
			// 创建值实例来分析字段
			valueInstance := reflect.New(valueType).Interface()
			// 生成值字段信息，使用占位符格式
			valueFields := extractFieldInfo(valueInstance, fullPath+".{KEY}", envPrefix, cmdPrefix)
			fields = append(fields, valueFields...)
		} else {
			// 基本类型 map，作为叶子节点处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix))
		}

	case reflect.Ptr:
		// 指针类型，获取指向的类型
		if fieldValue.IsNil() {
			// 创建一个新实例来分析结构
			newValue := reflect.New(fieldValue.Type().Elem())
			nestedFields := extractFieldsRecursive(newValue.Elem(), field, fullPath, helpText, envPrefix, cmdPrefix)
			fields = append(fields, nestedFields...)
		} else {
			nestedFields := extractFieldsRecursive(fieldValue.Elem(), field, fullPath, helpText, envPrefix, cmdPrefix)
			fields = append(fields, nestedFields...)
		}

	default:
		// 基本类型，作为叶子节点处理
		fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix))
	}

	return fields
}

// getFieldConfigName 获取字段的配置名称
func getFieldConfigName(field reflect.StructField) string {
	// 按优先级检查标签：cfg > json > yaml > toml > ini > 字段名
	tags := []string{"cfg", "json", "yaml", "toml", "ini"}

	for _, tag := range tags {
		if tagValue := field.Tag.Get(tag); tagValue != "" {
			tagName := strings.Split(tagValue, ",")[0]
			if tagName != "" {
				return tagName
			}
		}
	}

	return field.Name
}

// createFieldInfo 创建字段信息
func createFieldInfo(path string, field reflect.StructField, help, envPrefix, cmdPrefix string) FieldInfo {
	fieldType := getTypeName(field.Type)

	// 生成环境变量名
	envName := generateEnvName(path, envPrefix)

	// 生成命令行参数名
	cmdName := generateCmdName(path, cmdPrefix)

	// 获取示例值
	examples := generateExamples(field.Type)

	// 检查是否必填
	required := strings.Contains(field.Tag.Get("validate"), "required") ||
		strings.Contains(field.Tag.Get("binding"), "required")

	return FieldInfo{
		Path:     path,
		Type:     fieldType,
		Help:     help,
		EnvName:  envName,
		CmdName:  cmdName,
		Required: required,
		Examples: examples,
	}
}

// generateEnvName 生成环境变量名
func generateEnvName(path, prefix string) string {
	// 处理占位符：[N] -> _N_, {KEY} -> _{KEY}_
	envPath := strings.ReplaceAll(path, "[N]", "_N")
	envPath = strings.ReplaceAll(envPath, ".{KEY}", "_{KEY}")
	envPath = strings.ReplaceAll(envPath, "[0]", "_0") // 兼容已有的 [0] 格式

	envName := strings.ToUpper(strings.ReplaceAll(envPath, ".", "_"))
	if prefix != "" {
		return prefix + envName
	}
	return envName
}

// generateCmdName 生成命令行参数名
func generateCmdName(path, prefix string) string {
	// 处理占位符：[N] -> -N-, {KEY} -> -{KEY}-
	cmdPath := strings.ReplaceAll(path, "[N]", "-N")
	cmdPath = strings.ReplaceAll(cmdPath, ".{KEY}", "-{KEY}")
	cmdPath = strings.ReplaceAll(cmdPath, "[0]", "-0") // 兼容已有的 [0] 格式

	cmdName := strings.ToLower(strings.ReplaceAll(cmdPath, ".", "-"))
	if prefix != "" {
		return "--" + prefix + cmdName
	}
	return "--" + cmdName
}

// getTypeName 获取类型名称
func getTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + getTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + getTypeName(t.Elem())
	case reflect.Map:
		return "map[" + getTypeName(t.Key()) + "]" + getTypeName(t.Elem())
	case reflect.Struct:
		if isTimeType(t) {
			if t == reflect.TypeOf(time.Time{}) {
				return "time.Time"
			}
			if t == reflect.TypeOf(time.Duration(0)) {
				return "time.Duration"
			}
		}
		return t.Name()
	default:
		return t.String()
	}
}

// isTimeType 检查是否为时间类型
func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(time.Duration(0))
}

// generateDefaultHelp 生成默认帮助信息
func generateDefaultHelp(field reflect.StructField) string {
	fieldType := getTypeName(field.Type)
	return fmt.Sprintf("%s 类型的配置项", fieldType)
}

// generateExamples 生成示例值
func generateExamples(t reflect.Type) []string {
	// 首先检查特殊时间类型
	if t == reflect.TypeOf(time.Duration(0)) {
		return []string{"\"30s\"", "\"5m\"", "\"1h\""}
	}
	if t == reflect.TypeOf(time.Time{}) {
		return []string{"\"2023-12-25T15:30:45Z\"", "\"2023-12-25\"", "1703517045"}
	}

	switch t.Kind() {
	case reflect.String:
		return []string{"\"example-value\""}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{"123"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{"123"}
	case reflect.Float32, reflect.Float64:
		return []string{"3.14"}
	case reflect.Bool:
		return []string{"true", "false"}
	case reflect.Slice:
		elemType := getTypeName(t.Elem())
		return []string{fmt.Sprintf("使用数组索引: %s[0], %s[1], ...", elemType, elemType)}
	case reflect.Map:
		keyType := getTypeName(t.Key())
		valueType := getTypeName(t.Elem())
		return []string{fmt.Sprintf("使用键值对: %s_KEY1=%s, %s_KEY2=%s", keyType, valueType, keyType, valueType)}
	}

	return []string{"..."}
}







// formatFieldHelp 格式化字段帮助信息
func formatFieldHelp(field FieldInfo) string {
	var sb strings.Builder

	// 字段路径和类型
	sb.WriteString(fmt.Sprintf("  %s (%s)", field.Path, field.Type))
	if field.Required {
		sb.WriteString(" [必填]")
	}
	sb.WriteString("\n")

	// 帮助信息
	if field.Help != "" {
		sb.WriteString(fmt.Sprintf("    说明: %s\n", field.Help))
	}

	// 环境变量
	sb.WriteString(fmt.Sprintf("    环境变量: %s\n", field.EnvName))

	// 命令行参数
	sb.WriteString(fmt.Sprintf("    命令行参数: %s\n", field.CmdName))

	// 示例值
	if len(field.Examples) > 0 {
		sb.WriteString("    示例: ")
		sb.WriteString(strings.Join(field.Examples, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateTypeHelp 生成类型说明
func generateTypeHelp() string {
	return `类型说明：
  - string: 字符串类型
  - int/int64: 整数类型  
  - float64: 浮点数类型
  - bool: 布尔类型 (true/false)
  - time.Duration: 时间间隔 (如: 30s, 5m, 1h)
  - time.Time: 时间戳 (如: 2023-12-25T15:30:45Z, 1703517045)
  - []type: 数组类型，使用索引访问 (如: FIELD_0, FIELD_1)
  - map[string]type: 映射类型，使用键名访问 (如: FIELD_KEY1, FIELD_KEY2)

配置优先级 (从低到高):
  1. 配置文件
  2. 环境变量  
  3. 命令行参数

数组配置示例:
  环境变量: DATABASE_POOLS_0_HOST=db1.com, DATABASE_POOLS_1_HOST=db2.com
  命令行: --database-pools-0-host=db1.com --database-pools-1-host=db2.com

映射配置示例:
  环境变量: CACHE_REDIS_HOST=localhost, CACHE_MEMCACHED_HOST=mc.com
  命令行: --cache-redis-host=localhost --cache-memcached-host=mc.com
`
}
