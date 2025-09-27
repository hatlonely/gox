package rdb

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// TableModel 表模型定义
type TableModel struct {
	Table      string // 表名
	Fields     []FieldDefinition
	PrimaryKey []string          // 主键字段名列表，支持复合主键
	Indexes    []IndexDefinition // 普通索引
}

// FieldDefinition 字段定义
type FieldDefinition struct {
	Name     string
	Type     FieldType
	Required bool
	Default  any
	Size     int // 字段长度，如 VARCHAR(255)
}

// FieldType 字段类型
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeInt    FieldType = "int"
	FieldTypeFloat  FieldType = "float"
	FieldTypeBool   FieldType = "bool"
	FieldTypeDate   FieldType = "date"
	FieldTypeJSON   FieldType = "json"
)

// IndexDefinition 索引定义
type IndexDefinition struct {
	Name   string
	Fields []string
	Unique bool
}

// TableModelBuilder 表模型构建器
type TableModelBuilder struct{}

// NewTableModelBuilder 创建新的表模型构建器
func NewTableModelBuilder() *TableModelBuilder {
	return &TableModelBuilder{}
}

// FromStruct 从结构体构建 TableModel
// 支持的 tag 格式：
// - `rdb:"column_name,type=string,size=255,required,primary,index,unique"`
// - `table:"table_name"` 用于指定表名（在结构体级别）
func (b *TableModelBuilder) FromStruct(v any) (*TableModel, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", v)
	}

	rt := rv.Type()

	// 获取表名
	tableName := b.getTableName(rt)
	if tableName == "" {
		// 如果没有指定表名，使用结构体名的小写形式
		tableName = strings.ToLower(rt.Name())
	}

	model := &TableModel{
		Table:   tableName,
		Fields:  []FieldDefinition{},
		Indexes: []IndexDefinition{},
	}

	var primaryKeys []string
	indexMap := make(map[string]*IndexDefinition)

	// 遍历结构体字段
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		// 解析 rdb tag
		rdbTag := field.Tag.Get("rdb")
		if rdbTag == "-" {
			continue // 跳过被忽略的字段
		}

		fieldDef, isPrimary, indexes, err := b.parseFieldTag(field, rdbTag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %v", field.Name, err)
		}

		model.Fields = append(model.Fields, fieldDef)

		// 处理主键
		if isPrimary {
			primaryKeys = append(primaryKeys, fieldDef.Name)
		}

		// 处理索引
		for _, idx := range indexes {
			if existing, exists := indexMap[idx.Name]; exists {
				// 合并字段到现有索引
				existing.Fields = append(existing.Fields, fieldDef.Name)
			} else {
				idx.Fields = []string{fieldDef.Name}
				indexMap[idx.Name] = &idx
			}
		}
	}

	model.PrimaryKey = primaryKeys

	// 添加索引到模型
	for _, idx := range indexMap {
		model.Indexes = append(model.Indexes, *idx)
	}

	return model, nil
}

// getTableName 从结构体类型获取表名
func (b *TableModelBuilder) getTableName(rt reflect.Type) string {
	// 检查是否有 table tag
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if tableTag := field.Tag.Get("table"); tableTag != "" {
			return tableTag
		}
	}

	// 检查结构体本身是否有 table tag（通过匿名字段实现）
	if rt.Kind() == reflect.Struct {
		// 这里可以扩展支持结构体级别的 tag
		// 目前使用结构体名称的小写形式
	}

	return ""
}

// parseFieldTag 解析字段的 rdb tag
func (b *TableModelBuilder) parseFieldTag(field reflect.StructField, tag string) (FieldDefinition, bool, []IndexDefinition, error) {
	fieldDef := FieldDefinition{
		Name: field.Name, // 默认使用字段名
		Type: b.inferFieldType(field.Type),
	}

	var isPrimary bool
	var indexes []IndexDefinition

	if tag == "" {
		return fieldDef, isPrimary, indexes, nil
	}

	// 解析 tag 参数
	parts := strings.Split(tag, ",")

	// 第一部分是字段名（如果指定）
	if parts[0] != "" && !strings.Contains(parts[0], "=") {
		fieldDef.Name = parts[0]
		parts = parts[1:]
	}

	// 解析其他参数
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "=") {
			// 键值对参数
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "type":
				fieldDef.Type = FieldType(value)
			case "size":
				if size, err := strconv.Atoi(value); err == nil {
					fieldDef.Size = size
				}
			case "default":
				fieldDef.Default = b.parseDefaultValue(value, fieldDef.Type)
			case "index":
				// 指定索引名
				indexes = append(indexes, IndexDefinition{
					Name:   value,
					Unique: false,
				})
			case "unique":
				// 指定唯一索引名
				indexes = append(indexes, IndexDefinition{
					Name:   value,
					Unique: true,
				})
			}
		} else {
			// 布尔参数
			switch part {
			case "required", "not_null":
				fieldDef.Required = true
			case "primary", "pk":
				isPrimary = true
			case "index":
				// 创建默认索引名
				indexName := fmt.Sprintf("idx_%s", fieldDef.Name)
				indexes = append(indexes, IndexDefinition{
					Name:   indexName,
					Unique: false,
				})
			case "unique":
				// 创建默认唯一索引名
				indexName := fmt.Sprintf("uk_%s", fieldDef.Name)
				indexes = append(indexes, IndexDefinition{
					Name:   indexName,
					Unique: true,
				})
			}
		}
	}

	return fieldDef, isPrimary, indexes, nil
}

// inferFieldType 从 Go 类型推断字段类型
func (b *TableModelBuilder) inferFieldType(t reflect.Type) FieldType {
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return FieldTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FieldTypeInt
	case reflect.Float32, reflect.Float64:
		return FieldTypeFloat
	case reflect.Bool:
		return FieldTypeBool
	default:
		// 检查是否是时间类型
		if t.String() == "time.Time" {
			return FieldTypeDate
		}
		// 其他复杂类型默认为 JSON
		return FieldTypeJSON
	}
}

// parseDefaultValue 解析默认值
func (b *TableModelBuilder) parseDefaultValue(value string, fieldType FieldType) any {
	switch fieldType {
	case FieldTypeString:
		// 去掉引号
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
		return value
	case FieldTypeInt:
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
		return 0
	case FieldTypeFloat:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
		return 0.0
	case FieldTypeBool:
		return value == "true" || value == "1"
	default:
		return value
	}
}
