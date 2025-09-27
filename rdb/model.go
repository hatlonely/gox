package rdb

// TableModel 表模型定义
type TableModel struct {
	Table      string            // 表名
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