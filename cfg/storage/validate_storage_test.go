package storage

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的结构体 - 带校验标签
type TestValidationStruct struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"required,min=18,max=120"`
}

// 测试用的结构体 - 无校验标签
type TestNoValidationStruct struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// 测试数据集
var testValidationData = map[string]interface{}{
	"database": map[string]interface{}{
		"host": "localhost",
		"port": 3306,
	},
	"user": map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   25,
	},
	"invalid_user": map[string]interface{}{
		"name":  "J",  // 太短，不符合 min=2
		"email": "invalid-email",
		"age":   15,  // 太小，不符合 min=18
	},
}

// TestValidateStorage_Creation 测试 ValidateStorage 的创建
func TestValidateStorage_Creation(t *testing.T) {
	Convey("ValidateStorage 创建测试", t, func() {
		Convey("使用有效的 storage 创建", func() {
			baseStorage := NewMapStorage(testValidationData)
			validateStorage := NewValidateStorage(baseStorage)
			
			So(validateStorage, ShouldNotBeNil)
			So(validateStorage.storage, ShouldEqual, baseStorage)
		})

		Convey("使用 nil storage 创建", func() {
			validateStorage := NewValidateStorage(nil)
			
			So(validateStorage, ShouldNotBeNil)
			So(validateStorage.storage, ShouldBeNil)
		})
	})
}

// TestValidateStorage_Sub 测试 Sub 方法
func TestValidateStorage_Sub(t *testing.T) {
	Convey("ValidateStorage Sub 方法测试", t, func() {
		baseStorage := NewMapStorage(testValidationData)
		validateStorage := NewValidateStorage(baseStorage)

		Convey("获取存在的子配置", func() {
			subStorage := validateStorage.Sub("user")
			
			So(subStorage, ShouldNotBeNil)
			// 应该返回包装后的 ValidateStorage
			validateSub, ok := subStorage.(*ValidateStorage)
			So(ok, ShouldBeTrue)
			So(validateSub.storage, ShouldNotBeNil)
		})

		Convey("获取不存在的子配置", func() {
			subStorage := validateStorage.Sub("nonexistent")
			
			So(subStorage, ShouldNotBeNil)
			// 应该返回包装后的 ValidateStorage，但内部 storage 为 nil
			validateSub, ok := subStorage.(*ValidateStorage)
			So(ok, ShouldBeTrue)
			So(validateSub.storage, ShouldBeNil)
		})

		Convey("当 storage 为 nil 时调用 Sub", func() {
			validateStorage := NewValidateStorage(nil)
			subStorage := validateStorage.Sub("any")
			
			So(subStorage, ShouldBeNil)
		})
	})
}

// TestValidateStorage_ConvertTo 测试 ConvertTo 方法
func TestValidateStorage_ConvertTo(t *testing.T) {
	Convey("ValidateStorage ConvertTo 方法测试", t, func() {
		baseStorage := NewMapStorage(testValidationData)
		validateStorage := NewValidateStorage(baseStorage)

		Convey("转换到有效的结构体（通过验证）", func() {
			userStorage := validateStorage.Sub("user")
			var user TestValidationStruct
			
			err := userStorage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 25)
		})

		Convey("转换到无效的结构体（验证失败）", func() {
			invalidUserStorage := validateStorage.Sub("invalid_user")
			var user TestValidationStruct
			
			err := invalidUserStorage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "validation failed")
		})

		Convey("转换到无验证标签的结构体", func() {
			userStorage := validateStorage.Sub("user")
			var user TestNoValidationStruct
			
			err := userStorage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 25)
		})

		Convey("基础 storage 转换失败", func() {
			// 创建一个会导致转换失败的情况
			invalidData := map[string]interface{}{
				"age": "not-a-number", // 字符串不能转换为 int
			}
			baseStorage := NewMapStorage(invalidData)
			validateStorage := NewValidateStorage(baseStorage)
			
			var user TestValidationStruct
			err := validateStorage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			// 可能是转换错误或验证错误，我们只要确保有错误即可
		})

		Convey("当 storage 为 nil 时调用 ConvertTo", func() {
			validateStorage := NewValidateStorage(nil)
			var user TestValidationStruct
			
			err := validateStorage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			// user 应该保持原始状态
			So(user.Name, ShouldEqual, "")
			So(user.Email, ShouldEqual, "")
			So(user.Age, ShouldEqual, 0)
		})
	})
}

// TestValidateStorage_Equals 测试 Equals 方法
func TestValidateStorage_Equals(t *testing.T) {
	Convey("ValidateStorage Equals 方法测试", t, func() {
		baseStorage1 := NewMapStorage(testValidationData)
		baseStorage2 := NewMapStorage(testValidationData)
		differentStorage := NewMapStorage(map[string]interface{}{"key": "value"})

		Convey("相同内容的 ValidateStorage 应该相等", func() {
			validateStorage1 := NewValidateStorage(baseStorage1)
			validateStorage2 := NewValidateStorage(baseStorage2)
			
			So(validateStorage1.Equals(validateStorage2), ShouldBeTrue)
		})

		Convey("不同内容的 ValidateStorage 应该不相等", func() {
			validateStorage1 := NewValidateStorage(baseStorage1)
			validateStorage2 := NewValidateStorage(differentStorage)
			
			So(validateStorage1.Equals(validateStorage2), ShouldBeFalse)
		})

		Convey("ValidateStorage 与原始 Storage 比较", func() {
			validateStorage := NewValidateStorage(baseStorage1)
			
			So(validateStorage.Equals(baseStorage2), ShouldBeTrue)
			So(validateStorage.Equals(differentStorage), ShouldBeFalse)
		})

		Convey("nil storage 的比较", func() {
			validateStorage1 := NewValidateStorage(nil)
			validateStorage2 := NewValidateStorage(nil)
			validateStorageWithData := NewValidateStorage(baseStorage1)
			
			// 两个都是 ValidateStorage，即使内部 storage 都是 nil，也不相等
			// 因为 validateStorage2 本身不是 nil，所以 other == nil 返回 false
			So(validateStorage1.Equals(validateStorage2), ShouldBeFalse)
			// nil ValidateStorage 与真正的 nil 比较为 true
			So(validateStorage1.Equals(nil), ShouldBeTrue)
			So(validateStorage1.Equals(validateStorageWithData), ShouldBeFalse)
		})

		Convey("与 nil 比较", func() {
			validateStorage := NewValidateStorage(baseStorage1)
			
			So(validateStorage.Equals(nil), ShouldBeFalse)
		})
	})
}