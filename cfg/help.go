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

	// 按照层级分组显示
	groupedFields := groupFieldsByCategory(fields)

	for category, categoryFields := range groupedFields {
		if category != "" {
			sb.WriteString(fmt.Sprintf("=== %s ===\n", category))
		}

		for _, field := range categoryFields {
			sb.WriteString(formatFieldHelp(field))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// 添加类型说明
	sb.WriteString(generateTypeHelp())

	return sb.String()
}

// extractFieldInfo 提取字段信息
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

		// 处理不同类型的字段
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
			// 切片类型
			sliceHelp := fmt.Sprintf("%s (数组类型)", helpText)
			if elemType := fieldValue.Type().Elem(); elemType.Kind() == reflect.Struct {
				sliceHelp += fmt.Sprintf("\n    元素类型: %s", getTypeName(elemType))
				// 为切片元素提供示例结构
				if fieldValue.Len() > 0 || elemType.Kind() == reflect.Struct {
					sliceHelp += generateSliceElementHelp(elemType, fullPath, envPrefix, cmdPrefix)
				}
			}
			fields = append(fields, createFieldInfoWithCustomHelp(fullPath, field, sliceHelp, envPrefix, cmdPrefix))

		case reflect.Map:
			// Map 类型
			mapHelp := fmt.Sprintf("%s (映射类型)", helpText)
			keyType := fieldValue.Type().Key()
			valueType := fieldValue.Type().Elem()
			mapHelp += fmt.Sprintf("\n    键类型: %s, 值类型: %s", getTypeName(keyType), getTypeName(valueType))
			mapHelp += generateMapHelp(fieldValue.Type(), fullPath, envPrefix, cmdPrefix)
			fields = append(fields, createFieldInfoWithCustomHelp(fullPath, field, mapHelp, envPrefix, cmdPrefix))

		case reflect.Ptr:
			// 指针类型，获取指向的类型
			if fieldValue.IsNil() {
				// 创建一个新实例来分析结构
				newValue := reflect.New(fieldValue.Type().Elem())
				nestedFields := extractFieldInfo(newValue.Interface(), fullPath, envPrefix, cmdPrefix)
				fields = append(fields, nestedFields...)
			} else {
				nestedFields := extractFieldInfo(fieldValue.Interface(), fullPath, envPrefix, cmdPrefix)
				fields = append(fields, nestedFields...)
			}

		default:
			// 基本类型
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix))
		}
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
	return createFieldInfoWithCustomHelp(path, field, help, envPrefix, cmdPrefix)
}

// createFieldInfoWithCustomHelp 创建带自定义帮助的字段信息
func createFieldInfoWithCustomHelp(path string, field reflect.StructField, help, envPrefix, cmdPrefix string) FieldInfo {
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
	envName := strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
	if prefix != "" {
		return prefix + envName
	}
	return envName
}

// generateCmdName 生成命令行参数名
func generateCmdName(path, prefix string) string {
	cmdName := strings.ToLower(strings.ReplaceAll(path, ".", "-"))
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

// generateSliceElementHelp 生成切片元素帮助信息
func generateSliceElementHelp(elemType reflect.Type, path, envPrefix, cmdPrefix string) string {
	var help strings.Builder
	help.WriteString("\n    数组元素配置格式:")

	// 环境变量格式
	envName := generateEnvName(path, envPrefix)
	help.WriteString(fmt.Sprintf("\n      环境变量: %s_0_FIELD, %s_1_FIELD, ...", envName, envName))

	// 命令行格式
	cmdName := generateCmdName(path, cmdPrefix)
	if strings.HasPrefix(cmdName, "--") {
		cmdName = cmdName[2:]
	}
	help.WriteString(fmt.Sprintf("\n      命令行: --%s-0-field, --%s-1-field, ...", cmdName, cmdName))

	return help.String()
}

// generateMapHelp 生成 Map 帮助信息
func generateMapHelp(mapType reflect.Type, path, envPrefix, cmdPrefix string) string {
	var help strings.Builder
	help.WriteString("\n    映射配置格式:")

	// 环境变量格式
	envName := generateEnvName(path, envPrefix)
	help.WriteString(fmt.Sprintf("\n      环境变量: %s_KEY1=value1, %s_KEY2=value2, ...", envName, envName))

	// 命令行格式
	cmdName := generateCmdName(path, cmdPrefix)
	if strings.HasPrefix(cmdName, "--") {
		cmdName = cmdName[2:]
	}
	help.WriteString(fmt.Sprintf("\n      命令行: --%s-key1=value1, --%s-key2=value2, ...", cmdName, cmdName))

	return help.String()
}

// groupFieldsByCategory 按类别分组字段
func groupFieldsByCategory(fields []FieldInfo) map[string][]FieldInfo {
	groups := make(map[string][]FieldInfo)

	for _, field := range fields {
		parts := strings.Split(field.Path, ".")
		category := ""
		if len(parts) > 1 {
			category = parts[0]
		}

		groups[category] = append(groups[category], field)
	}

	return groups
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
