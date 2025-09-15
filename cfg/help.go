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
	Validation   string   // 校验信息，从 validate 标签获取
	Order        int      // 字段定义顺序，用于保持原始顺序
}

// GenerateHelp 生成配置帮助信息
// config: 配置结构体实例
// envPrefix: 环境变量前缀，如 "APP_"
// cmdPrefix: 命令行参数前缀，如 "app-"
func GenerateHelp(config interface{}, envPrefix, cmdPrefix string) string {
	orderCounter := &orderCounter{}
	fields := extractFieldInfo(config, "", envPrefix, cmdPrefix, orderCounter)

	if len(fields) == 0 {
		return "未找到配置字段信息"
	}

	var sb strings.Builder
	sb.WriteString("配置参数说明：\n\n")

	// 按原始定义顺序排序，保持结构体字段的原始顺序
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Order < fields[j].Order
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

// orderCounter 用于记录字段的原始顺序
type orderCounter struct {
	count int
}

func (oc *orderCounter) next() int {
	oc.count++
	return oc.count
}

// extractFieldInfo 提取字段信息，只在叶子节点生成 FieldInfo
func extractFieldInfo(obj interface{}, prefix, envPrefix, cmdPrefix string, orderCounter *orderCounter) []FieldInfo {
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
		fields = append(fields, extractFieldsRecursive(fieldValue, field, fullPath, helpText, envPrefix, cmdPrefix, orderCounter)...)
	}

	return fields
}

// extractFieldsRecursive 递归提取字段信息
func extractFieldsRecursive(fieldValue reflect.Value, field reflect.StructField, fullPath, helpText, envPrefix, cmdPrefix string, orderCounter *orderCounter) []FieldInfo {
	var fields []FieldInfo

	switch fieldValue.Kind() {
	case reflect.Struct:
		// 嵌套结构体，递归处理
		if isTimeType(fieldValue.Type()) {
			// time.Time 和 time.Duration 作为基本类型处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix, orderCounter))
		} else {
			// 递归处理嵌套结构体
			nestedFields := extractFieldInfo(fieldValue.Interface(), fullPath, envPrefix, cmdPrefix, orderCounter)
			fields = append(fields, nestedFields...)
		}

	case reflect.Slice:
		// 切片类型，递归处理元素字段
		if elemType := fieldValue.Type().Elem(); elemType.Kind() == reflect.Struct {
			// 创建元素实例来分析字段
			elemInstance := reflect.New(elemType).Interface()
			// 生成元素字段信息，使用占位符格式
			elemFields := extractFieldInfo(elemInstance, fullPath+"[N]", envPrefix, cmdPrefix, orderCounter)
			fields = append(fields, elemFields...)
		} else {
			// 基本类型数组，直接作为叶子节点处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix, orderCounter))
		}

	case reflect.Map:
		// Map类型，处理值类型
		valueType := fieldValue.Type().Elem()

		// 如果值类型是结构体，递归处理使用占位符 {KEY}
		if valueType.Kind() == reflect.Struct {
			// 创建值实例来分析字段
			valueInstance := reflect.New(valueType).Interface()
			// 生成值字段信息，使用占位符格式
			valueFields := extractFieldInfo(valueInstance, fullPath+".{KEY}", envPrefix, cmdPrefix, orderCounter)
			fields = append(fields, valueFields...)
		} else {
			// 基本类型 map，作为叶子节点处理
			fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix, orderCounter))
		}

	case reflect.Ptr:
		// 指针类型，获取指向的类型
		if fieldValue.IsNil() {
			// 创建一个新实例来分析结构
			newValue := reflect.New(fieldValue.Type().Elem())
			nestedFields := extractFieldsRecursive(newValue.Elem(), field, fullPath, helpText, envPrefix, cmdPrefix, orderCounter)
			fields = append(fields, nestedFields...)
		} else {
			nestedFields := extractFieldsRecursive(fieldValue.Elem(), field, fullPath, helpText, envPrefix, cmdPrefix, orderCounter)
			fields = append(fields, nestedFields...)
		}

	default:
		// 基本类型，作为叶子节点处理
		fields = append(fields, createFieldInfo(fullPath, field, helpText, envPrefix, cmdPrefix, orderCounter))
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
func createFieldInfo(path string, field reflect.StructField, help, envPrefix, cmdPrefix string, orderCounter *orderCounter) FieldInfo {
	fieldType := getTypeName(field.Type)

	// 生成环境变量名
	envName := generateEnvName(path, envPrefix)

	// 生成命令行参数名
	cmdName := generateCmdName(path, cmdPrefix)

	// 获取示例值：只有在有 eg 标签时才显示示例值
	var examples []string
	if exampleTag := field.Tag.Get("eg"); exampleTag != "" {
		// 使用标签中的示例值
		examples = []string{exampleTag}
	}
	// 如果没有 eg 标签，不显示示例值（examples 为空）

	// 获取默认值
	defaultValue := field.Tag.Get("def")

	// 获取校验信息
	validation := field.Tag.Get("validate")

	// 检查是否必填
	required := strings.Contains(validation, "required") ||
		strings.Contains(field.Tag.Get("binding"), "required")

	return FieldInfo{
		Path:         path,
		Type:         fieldType,
		Help:         help,
		EnvName:      envName,
		CmdName:      cmdName,
		Required:     required,
		Examples:     examples,
		DefaultValue: defaultValue,
		Validation:   validation,
		Order:        orderCounter.next(), // 记录字段定义的原始顺序
	}
}

// generateEnvName 生成环境变量名
func generateEnvName(path, prefix string) string {
	// 处理占位符：[N] -> _{N}, {KEY} -> _{KEY}
	envPath := strings.ReplaceAll(path, "[N]", "_{N}")
	envPath = strings.ReplaceAll(envPath, ".{KEY}", "_{KEY}")

	envName := strings.ToUpper(strings.ReplaceAll(envPath, ".", "_"))
	if prefix != "" {
		return prefix + envName
	}
	return envName
}

// generateCmdName 生成命令行参数名
func generateCmdName(path, prefix string) string {
	// 处理占位符：[N] -> -{n}, {KEY} -> -{key}
	cmdPath := strings.ReplaceAll(path, "[N]", "-{n}")
	cmdPath = strings.ReplaceAll(cmdPath, ".{KEY}", "-{key}")

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

	// 校验信息
	if field.Validation != "" {
		sb.WriteString(fmt.Sprintf("    校验规则: %s\n", formatValidationRules(field.Validation)))
	}

	// 环境变量
	sb.WriteString(fmt.Sprintf("    环境变量: %s\n", field.EnvName))

	// 命令行参数
	sb.WriteString(fmt.Sprintf("    命令行参数: %s\n", field.CmdName))

	// 默认值
	if field.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf("    默认值: %s\n", field.DefaultValue))
	}

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

// formatValidationRules 格式化校验规则信息
func formatValidationRules(validation string) string {
	if validation == "" {
		return ""
	}

	// 将校验规则转换为更易读的格式
	rules := strings.Split(validation, ",")
	var formatted []string

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// 解析校验规则
		formatted = append(formatted, formatSingleRule(rule))
	}

	if len(formatted) == 0 {
		return validation // 如果没有识别的规则，返回原始内容
	}

	return strings.Join(formatted, "; ")
}

// formatSingleRule 格式化单个校验规则
func formatSingleRule(rule string) string {
	// 处理带参数的规则
	if strings.Contains(rule, "=") {
		parts := strings.SplitN(rule, "=", 2)
		key := parts[0]
		value := parts[1]

		switch key {
		case "min":
			return fmt.Sprintf("最小值: %s", value)
		case "max":
			return fmt.Sprintf("最大值: %s", value)
		case "len":
			return fmt.Sprintf("长度: %s", value)
		case "gt":
			return fmt.Sprintf("大于: %s", value)
		case "gte":
			return fmt.Sprintf("大于等于: %s", value)
		case "lt":
			return fmt.Sprintf("小于: %s", value)
		case "lte":
			return fmt.Sprintf("小于等于: %s", value)
		case "eq":
			return fmt.Sprintf("等于: %s", value)
		case "ne":
			return fmt.Sprintf("不等于: %s", value)
		case "oneof":
			return fmt.Sprintf("允许值: %s", strings.ReplaceAll(value, " ", ", "))
		case "contains":
			return fmt.Sprintf("包含: %s", value)
		case "containsany":
			return fmt.Sprintf("包含任意字符: %s", value)
		case "excludes":
			return fmt.Sprintf("不包含: %s", value)
		case "startswith":
			return fmt.Sprintf("开头为: %s", value)
		case "endswith":
			return fmt.Sprintf("结尾为: %s", value)
		case "eqfield":
			return fmt.Sprintf("等于字段: %s", value)
		case "nefield":
			return fmt.Sprintf("不等于字段: %s", value)
		case "gtfield":
			return fmt.Sprintf("大于字段: %s", value)
		case "gtefield":
			return fmt.Sprintf("大于等于字段: %s", value)
		case "ltfield":
			return fmt.Sprintf("小于字段: %s", value)
		case "ltefield":
			return fmt.Sprintf("小于等于字段: %s", value)
		// 条件校验
		case "required_with":
			return fmt.Sprintf("与其他字段一起必填: %s", value)
		case "required_with_all":
			return fmt.Sprintf("与所有其他字段一起必填: %s", value)
		case "required_without":
			return fmt.Sprintf("缺少其他字段时必填: %s", value)
		case "required_without_all":
			return fmt.Sprintf("缺少所有其他字段时必填: %s", value)
		case "required_if":
			return fmt.Sprintf("条件必填: %s", value)
		case "required_unless":
			return fmt.Sprintf("除非必填: %s", value)
		default:
			return rule // 未识别的带参数规则，返回原始内容
		}
	}

	// 处理无参数的规则
	switch rule {
	// 基础校验
	case "required":
		return "必填"
	case "omitempty":
		return "可为空"

	// 比较
	case "eq_ignore_case":
		return "等于(忽略大小写)"
	case "ne_ignore_case":
		return "不等于(忽略大小写)"

	// 字符串类型
	case "alpha":
		return "仅允许字母"
	case "alphanum":
		return "仅允许字母数字"
	case "alphanumunicode":
		return "仅允许字母数字Unicode"
	case "alphaunicode":
		return "仅允许字母Unicode"
	case "ascii":
		return "仅允许ASCII字符"
	case "boolean":
		return "布尔值"
	case "lowercase":
		return "小写字母"
	case "uppercase":
		return "大写字母"
	case "number":
		return "数字"
	case "numeric":
		return "仅允许数字"
	case "printascii":
		return "可打印ASCII字符"
	case "multibyte":
		return "多字节字符"

	// 网络格式
	case "email":
		return "邮箱格式"
	case "url":
		return "URL格式"
	case "http_url":
		return "HTTP URL格式"
	case "uri":
		return "URI格式"
	case "hostname":
		return "主机名格式"
	case "hostname_rfc1123":
		return "主机名格式(RFC1123)"
	case "fqdn":
		return "完全限定域名"
	case "ip":
		return "IP地址"
	case "ipv4":
		return "IPv4地址"
	case "ipv6":
		return "IPv6地址"
	case "cidr":
		return "CIDR格式"
	case "cidrv4":
		return "CIDRv4格式"
	case "cidrv6":
		return "CIDRv6格式"
	case "mac":
		return "MAC地址"
	case "tcp_addr":
		return "TCP地址"
	case "tcp4_addr":
		return "TCPv4地址"
	case "tcp6_addr":
		return "TCPv6地址"
	case "udp_addr":
		return "UDP地址"
	case "udp4_addr":
		return "UDPv4地址"
	case "udp6_addr":
		return "UDPv6地址"
	case "unix_addr":
		return "Unix域套接字地址"

	// 格式校验
	case "base64":
		return "Base64编码"
	case "base64url":
		return "Base64URL编码"
	case "base64rawurl":
		return "Base64RawURL编码"
	case "json":
		return "JSON格式"
	case "jwt":
		return "JWT格式"
	case "uuid":
		return "UUID格式"
	case "uuid3":
		return "UUIDv3格式"
	case "uuid4":
		return "UUIDv4格式"
	case "uuid5":
		return "UUIDv5格式"
	case "ulid":
		return "ULID格式"
	case "credit_card":
		return "信用卡号码"
	case "isbn":
		return "ISBN格式"
	case "isbn10":
		return "ISBN10格式"
	case "isbn13":
		return "ISBN13格式"
	case "ssn":
		return "社会保障号"
	case "btc_addr":
		return "比特币地址"
	case "btc_addr_bech32":
		return "比特币Bech32地址"
	case "eth_addr":
		return "以太坊地址"
	case "hexadecimal":
		return "十六进制字符串"
	case "hexcolor":
		return "十六进制颜色"
	case "rgb":
		return "RGB颜色"
	case "rgba":
		return "RGBA颜色"
	case "hsl":
		return "HSL颜色"
	case "hsla":
		return "HSLA颜色"
	case "html":
		return "HTML标签"
	case "html_encoded":
		return "HTML编码"
	case "url_encoded":
		return "URL编码"

	// 哈希格式
	case "md4":
		return "MD4哈希"
	case "md5":
		return "MD5哈希"
	case "sha256":
		return "SHA256哈希"
	case "sha384":
		return "SHA384哈希"
	case "sha512":
		return "SHA512哈希"

	// 时间和版本
	case "datetime":
		return "日期时间格式"
	case "timezone":
		return "时区格式"
	case "semver":
		return "语义化版本"
	case "cron":
		return "Cron表达式"

	// 地理位置
	case "latitude":
		return "纬度"
	case "longitude":
		return "经度"

	// 国际化标准
	case "iso3166_1_alpha2":
		return "ISO3166-1国家代码(2位字母)"
	case "iso3166_1_alpha3":
		return "ISO3166-1国家代码(3位字母)"
	case "iso3166_1_alpha_numeric":
		return "ISO3166-1国家代码(数字)"
	case "iso3166_2":
		return "ISO3166-2地区代码"
	case "iso4217":
		return "ISO4217货币代码"
	case "bcp47_language_tag":
		return "BCP47语言标签"
	case "e164":
		return "E164电话号码"

	// 文件和路径
	case "file":
		return "文件路径"
	case "filepath":
		return "文件路径格式"
	case "dir":
		return "目录路径"
	case "dirpath":
		return "目录路径格式"
	case "image":
		return "图片格式"

	// 其他
	case "unique":
		return "唯一值"
	case "isdefault":
		return "默认值"
	case "iscolor":
		return "颜色格式"
	case "country_code":
		return "国家代码"
	case "luhn_checksum":
		return "Luhn校验码"
	case "mongodb":
		return "MongoDB ObjectID"
	case "cve":
		return "CVE标识符"

	// 条件校验
	case "required_if":
		return "条件必填"
	case "required_unless":
		return "除非必填"
	case "required_with":
		return "与其他字段一起必填"
	case "required_with_all":
		return "与所有其他字段一起必填"
	case "required_without":
		return "缺少其他字段时必填"
	case "required_without_all":
		return "缺少所有其他字段时必填"
	case "excluded_if":
		return "条件排除"
	case "excluded_unless":
		return "除非排除"
	case "excluded_with":
		return "与其他字段一起排除"
	case "excluded_with_all":
		return "与所有其他字段一起排除"
	case "excluded_without":
		return "缺少其他字段时排除"
	case "excluded_without_all":
		return "缺少所有其他字段时排除"

	default:
		// 对于未识别的规则，直接显示原始内容
		return rule
	}
}
