package database

import (
	"testing"
	"time"
)

// 测试用的结构体
type User struct {
	ID       int64     `rdb:"id,primary,type=int"`
	Username string    `rdb:"username,required,unique,size=50"`
	Email    string    `rdb:"email,required,unique=uk_user_email,size=100"`
	Password string    `rdb:"password,required,size=255"`
	Age      int       `rdb:"age,type=int,default=0"`
	IsActive bool      `rdb:"is_active,type=bool,default=true"`
	Profile  string    `rdb:"profile,type=json"`
	CreateAt time.Time `rdb:"created_at,type=date,required"`
	UpdateAt time.Time `rdb:"updated_at,type=date"`
	// 忽略的字段
	TempData string `rdb:"-"`
}

type Product struct {
	ID          int64   `rdb:"id,primary"`
	Name        string  `rdb:"name,required,size=100,index"`
	CategoryID  int64   `rdb:"category_id,required,index=idx_category"`
	Price       float64 `rdb:"price,required,type=float"`
	Description string  `rdb:"description,type=string,size=1000"`
	InStock     bool    `rdb:"in_stock,type=bool,default=true"`
}

// Order 测试联合索引的结构体
type Order struct {
	ID         int64  `rdb:"id,primary"`
	UserID     int64  `rdb:"user_id,required,index=idx_user_date"`
	OrderDate  string `rdb:"order_date,required,index=idx_user_date"`
	Status     string `rdb:"status,required,index=idx_status_date"`
	CreateDate string `rdb:"create_date,required,index=idx_status_date"`
	Amount     float64 `rdb:"amount,required"`
}

// CustomTableStruct 测试实现 Table() 方法的结构体
type CustomTableStruct struct {
	ID   int64  `rdb:"id,primary"`
	Name string `rdb:"name,required"`
}

// Table 实现自定义表名
func (CustomTableStruct) Table() string {
	return "custom_table_name"
}

func TestTableModelBuilder_FromStruct(t *testing.T) {
	builder := NewTableModelBuilder()

	t.Run("User struct", func(t *testing.T) {
		user := User{}
		model, err := builder.FromStruct(user)
		if err != nil {
			t.Fatalf("Failed to build model: %v", err)
		}

		// 验证表名
		expectedTable := "User"
		if model.Table != expectedTable {
			t.Errorf("Expected table name %s, got %s", expectedTable, model.Table)
		}

		// 验证字段数量（排除被忽略的字段）
		expectedFieldCount := 9 // id, username, email, password, age, is_active, profile, created_at, updated_at
		if len(model.Fields) != expectedFieldCount {
			t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(model.Fields))
		}

		// 验证主键
		expectedPK := []string{"id"}
		if len(model.PrimaryKey) != len(expectedPK) {
			t.Errorf("Expected primary key %v, got %v", expectedPK, model.PrimaryKey)
		} else {
			for i, pk := range expectedPK {
				if model.PrimaryKey[i] != pk {
					t.Errorf("Expected primary key %s, got %s", pk, model.PrimaryKey[i])
				}
			}
		}

		// 验证字段定义
		fieldMap := make(map[string]FieldDefinition)
		for _, field := range model.Fields {
			fieldMap[field.Name] = field
		}

		// 验证 ID 字段
		if idField, exists := fieldMap["id"]; exists {
			if idField.Type != FieldTypeInt {
				t.Errorf("Expected ID field type %s, got %s", FieldTypeInt, idField.Type)
			}
		} else {
			t.Error("ID field not found")
		}

		// 验证 username 字段
		if usernameField, exists := fieldMap["username"]; exists {
			if usernameField.Type != FieldTypeString {
				t.Errorf("Expected username field type %s, got %s", FieldTypeString, usernameField.Type)
			}
			if usernameField.Size != 50 {
				t.Errorf("Expected username field size 50, got %d", usernameField.Size)
			}
			if !usernameField.Required {
				t.Error("Expected username field to be required")
			}
		} else {
			t.Error("Username field not found")
		}

		// 验证 age 字段的默认值
		if ageField, exists := fieldMap["age"]; exists {
			if ageField.Default != 0 {
				t.Errorf("Expected age default value 0, got %v", ageField.Default)
			}
		}

		// 验证 is_active 字段的默认值
		if activeField, exists := fieldMap["is_active"]; exists {
			if activeField.Default != true {
				t.Errorf("Expected is_active default value true, got %v", activeField.Default)
			}
		}

		// 验证索引
		indexMap := make(map[string]IndexDefinition)
		for _, index := range model.Indexes {
			indexMap[index.Name] = index
		}

		// 验证 username 的唯一索引
		if usernameIndex, exists := indexMap["uk_username"]; exists {
			if !usernameIndex.Unique {
				t.Error("Expected username index to be unique")
			}
			if len(usernameIndex.Fields) != 1 || usernameIndex.Fields[0] != "username" {
				t.Errorf("Expected username index fields [username], got %v", usernameIndex.Fields)
			}
		} else {
			t.Error("Username unique index not found")
		}

		// 验证 email 的自定义唯一索引
		if emailIndex, exists := indexMap["uk_user_email"]; exists {
			if !emailIndex.Unique {
				t.Error("Expected email index to be unique")
			}
		} else {
			t.Error("Email unique index not found")
		}

		t.Logf("Generated model: %+v", model)
		for _, field := range model.Fields {
			t.Logf("Field: %+v", field)
		}
		for _, index := range model.Indexes {
			t.Logf("Index: %+v", index)
		}
	})

	t.Run("Product struct", func(t *testing.T) {
		product := Product{}
		model, err := builder.FromStruct(product)
		if err != nil {
			t.Fatalf("Failed to build model: %v", err)
		}

		// 验证表名
		expectedTable := "Product"
		if model.Table != expectedTable {
			t.Errorf("Expected table name %s, got %s", expectedTable, model.Table)
		}

		// 验证字段数量
		expectedFieldCount := 6
		if len(model.Fields) != expectedFieldCount {
			t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(model.Fields))
		}

		// 验证索引
		indexMap := make(map[string]IndexDefinition)
		for _, index := range model.Indexes {
			indexMap[index.Name] = index
		}

		// 验证 name 字段的普通索引
		if nameIndex, exists := indexMap["idx_name"]; exists {
			if nameIndex.Unique {
				t.Error("Expected name index to be non-unique")
			}
		} else {
			t.Error("Name index not found")
		}

		// 验证 category_id 的自定义索引
		if categoryIndex, exists := indexMap["idx_category"]; exists {
			if categoryIndex.Unique {
				t.Error("Expected category index to be non-unique")
			}
		} else {
			t.Error("Category index not found")
		}

		t.Logf("Generated model: %+v", model)
	})

	t.Run("Order struct with composite indexes", func(t *testing.T) {
		order := Order{}
		model, err := builder.FromStruct(order)
		if err != nil {
			t.Fatalf("Failed to build model: %v", err)
		}

		// 验证表名
		expectedTable := "Order"
		if model.Table != expectedTable {
			t.Errorf("Expected table name %s, got %s", expectedTable, model.Table)
		}

		// 验证字段数量
		expectedFieldCount := 6
		if len(model.Fields) != expectedFieldCount {
			t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(model.Fields))
		}

		// 验证索引
		indexMap := make(map[string]IndexDefinition)
		for _, index := range model.Indexes {
			indexMap[index.Name] = index
		}

		// 验证联合索引 idx_user_date (user_id + order_date)
		if userDateIndex, exists := indexMap["idx_user_date"]; exists {
			if userDateIndex.Unique {
				t.Error("Expected user_date index to be non-unique")
			}
			expectedFields := []string{"user_id", "order_date"}
			if len(userDateIndex.Fields) != len(expectedFields) {
				t.Errorf("Expected user_date index fields %v, got %v", expectedFields, userDateIndex.Fields)
			} else {
				for i, field := range expectedFields {
					if userDateIndex.Fields[i] != field {
						t.Errorf("Expected user_date index field[%d] %s, got %s", i, field, userDateIndex.Fields[i])
					}
				}
			}
		} else {
			t.Error("User date composite index not found")
		}

		// 验证联合索引 idx_status_date (status + create_date)
		if statusDateIndex, exists := indexMap["idx_status_date"]; exists {
			if statusDateIndex.Unique {
				t.Error("Expected status_date index to be non-unique")
			}
			expectedFields := []string{"status", "create_date"}
			if len(statusDateIndex.Fields) != len(expectedFields) {
				t.Errorf("Expected status_date index fields %v, got %v", expectedFields, statusDateIndex.Fields)
			} else {
				for i, field := range expectedFields {
					if statusDateIndex.Fields[i] != field {
						t.Errorf("Expected status_date index field[%d] %s, got %s", i, field, statusDateIndex.Fields[i])
					}
				}
			}
		} else {
			t.Error("Status date composite index not found")
		}

		// 验证索引总数应该是 2 个（两个联合索引）
		expectedIndexCount := 2
		if len(model.Indexes) != expectedIndexCount {
			t.Errorf("Expected %d indexes, got %d", expectedIndexCount, len(model.Indexes))
		}

		t.Logf("Generated model with composite indexes: %+v", model)
		for _, index := range model.Indexes {
			t.Logf("Composite Index: %+v", index)
		}
	})

	t.Run("CustomTableStruct with Table() method", func(t *testing.T) {
		custom := CustomTableStruct{}
		model, err := builder.FromStruct(custom)
		if err != nil {
			t.Fatalf("Failed to build model: %v", err)
		}

		// 验证表名使用 Table() 方法返回的值
		expectedTable := "custom_table_name"
		if model.Table != expectedTable {
			t.Errorf("Expected table name %s, got %s", expectedTable, model.Table)
		}

		// 验证字段数量
		expectedFieldCount := 2
		if len(model.Fields) != expectedFieldCount {
			t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(model.Fields))
		}

		t.Logf("Generated model with custom table name: %+v", model)
	})
}

func TestTableModelBuilder_FieldTypeInference(t *testing.T) {
	builder := NewTableModelBuilder()

	type TestStruct struct {
		StringField string
		IntField    int
		Int64Field  int64
		FloatField  float64
		BoolField   bool
		TimeField   time.Time
		PtrField    *string
		SliceField  []string
	}

	test := TestStruct{}
	model, err := builder.FromStruct(test)
	if err != nil {
		t.Fatalf("Failed to build model: %v", err)
	}

	fieldMap := make(map[string]FieldDefinition)
	for _, field := range model.Fields {
		fieldMap[field.Name] = field
	}

	tests := []struct {
		fieldName    string
		expectedType FieldType
	}{
		{"StringField", FieldTypeString},
		{"IntField", FieldTypeInt},
		{"Int64Field", FieldTypeInt},
		{"FloatField", FieldTypeFloat},
		{"BoolField", FieldTypeBool},
		{"TimeField", FieldTypeDate},
		{"PtrField", FieldTypeString},
		{"SliceField", FieldTypeJSON},
	}

	for _, test := range tests {
		if field, exists := fieldMap[test.fieldName]; exists {
			if field.Type != test.expectedType {
				t.Errorf("Field %s: expected type %s, got %s",
					test.fieldName, test.expectedType, field.Type)
			}
		} else {
			t.Errorf("Field %s not found", test.fieldName)
		}
	}
}

func TestTableModelBuilder_ErrorCases(t *testing.T) {
	builder := NewTableModelBuilder()

	t.Run("Non-struct input", func(t *testing.T) {
		_, err := builder.FromStruct("not a struct")
		if err == nil {
			t.Error("Expected error for non-struct input")
		}
	})

	t.Run("Nil pointer", func(t *testing.T) {
		var user *User
		_, err := builder.FromStruct(user)
		if err == nil {
			t.Error("Expected error for nil pointer")
		}
	})
}
