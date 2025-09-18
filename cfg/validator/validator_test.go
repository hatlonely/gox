package validator

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValidateStruct(t *testing.T) {
	Convey("Validator 结构体校验测试", t, func() {
		
		// 定义测试用的结构体
		type User struct {
			Name  string `validate:"required,min=2,max=50"`
			Email string `validate:"required,email"`
			Age   int    `validate:"min=0,max=150"`
		}

		type Address struct {
			Street string `validate:"required"`
			City   string `validate:"required"`
		}

		type UserWithAddress struct {
			Name    string  `validate:"required"`
			Email   string  `validate:"required,email"`
			Address Address `validate:"required"`
		}

		Convey("有效的结构体校验", func() {
			user := User{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			}
			
			err := ValidateStruct(&user)
			So(err, ShouldBeNil)
		})

		Convey("校验失败 - 必填字段为空", func() {
			user := User{
				Name:  "",
				Email: "john@example.com",
				Age:   30,
			}
			
			err := ValidateStruct(&user)
			So(err, ShouldNotBeNil)
		})

		Convey("校验失败 - 邮箱格式错误", func() {
			user := User{
				Name:  "John Doe",
				Email: "invalid-email",
				Age:   30,
			}
			
			err := ValidateStruct(&user)
			So(err, ShouldNotBeNil)
		})

		Convey("嵌套结构体校验 - 有效", func() {
			user := UserWithAddress{
				Name:  "John Doe",
				Email: "john@example.com",
				Address: Address{
					Street: "123 Main St",
					City:   "New York",
				},
			}
			
			err := ValidateStruct(&user)
			So(err, ShouldBeNil)
		})

		Convey("嵌套结构体校验 - 失败", func() {
			user := UserWithAddress{
				Name:  "John Doe",
				Email: "john@example.com",
				Address: Address{
					Street: "",  // 必填字段为空
					City:   "New York",
				},
			}
			
			err := ValidateStruct(&user)
			So(err, ShouldNotBeNil)
		})

		Convey("nil 对象跳过校验", func() {
			err := ValidateStruct(nil)
			So(err, ShouldBeNil)
		})

		Convey("nil 指针跳过校验", func() {
			var user *User = nil
			err := ValidateStruct(&user)
			So(err, ShouldBeNil)
		})

		Convey("time.Time 类型跳过校验", func() {
			timeValue := time.Now()
			err := ValidateStruct(&timeValue)
			So(err, ShouldBeNil)
		})

		Convey("基本类型跳过校验", func() {
			intValue := 42
			err := ValidateStruct(&intValue)
			So(err, ShouldBeNil)
		})

		Convey("map 类型跳过校验", func() {
			mapValue := map[string]string{"key": "value"}
			err := ValidateStruct(&mapValue)
			So(err, ShouldBeNil)
		})

		Convey("slice 类型跳过校验", func() {
			sliceValue := []string{"item1", "item2"}
			err := ValidateStruct(&sliceValue)
			So(err, ShouldBeNil)
		})
	})
}