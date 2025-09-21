package parser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSeparatorLineParserWithOptions(t *testing.T) {
	Convey("NewSeparatorLineParserWithOptions", t, func() {
		Convey("创建基本SeparatorLineParser", func() {
			options := &LineParserOptions{
				Separator: "\t",
			}
			parser, err := NewLineParserWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.separator, ShouldEqual, "\t")
		})

		Convey("支持不同数据类型", func() {
			options := &LineParserOptions{
				Separator: ",",
			}

			Convey("string-int类型", func() {
				parser, err := NewLineParserWithOptions[string, int](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
				So(parser.separator, ShouldEqual, ",")
			})

			Convey("int-string类型", func() {
				parser, err := NewLineParserWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
				So(parser.separator, ShouldEqual, ",")
			})
		})

		Convey("自定义分隔符", func() {
			options := &LineParserOptions{
				Separator: "|",
			}
			parser, err := NewLineParserWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.separator, ShouldEqual, "|")
		})
	})
}

func TestSeparatorLineParserParse(t *testing.T) {
	Convey("SeparatorLineParser.Parse", t, func() {
		options := &LineParserOptions{
			Separator: "\t",
		}

		Convey("字符串类型解析", func() {
			parser, _ := NewLineParserWithOptions[string, string](options)

			Convey("基本key-value解析", func() {
				changeType, key, value, err := parser.Parse([]byte("hello\tworld"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "hello")
				So(value, ShouldEqual, "world")
			})

			Convey("带changeType的解析", func() {
				Convey("数值类型changeType", func() {
					changeType, key, value, err := parser.Parse([]byte("hello\tworld\t2"))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeUpdate)
					So(key, ShouldEqual, "hello")
					So(value, ShouldEqual, "world")
				})

				Convey("字符串类型changeType", func() {
					changeType, key, value, err := parser.Parse([]byte("hello\tworld\tdelete"))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeDelete)
					So(key, ShouldEqual, "hello")
					So(value, ShouldEqual, "world")
				})

				Convey("未知changeType默认为add", func() {
					changeType, key, value, err := parser.Parse([]byte("hello\tworld\tunknown"))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeAdd)
					So(key, ShouldEqual, "hello")
					So(value, ShouldEqual, "world")
				})
			})

			Convey("边界情况", func() {
				Convey("只有一个字段", func() {
					changeType, key, value, err := parser.Parse([]byte("hello"))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeUnknown)
					So(key, ShouldEqual, "")
					So(value, ShouldEqual, "")
				})

				Convey("空字符串", func() {
					changeType, key, value, err := parser.Parse([]byte(""))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeUnknown)
					So(key, ShouldEqual, "")
					So(value, ShouldEqual, "")
				})

				Convey("空的changeType字段", func() {
					changeType, key, value, err := parser.Parse([]byte("hello\tworld\t"))
					So(err, ShouldBeNil)
					So(changeType, ShouldEqual, ChangeTypeAdd)
					So(key, ShouldEqual, "hello")
					So(value, ShouldEqual, "world")
				})
			})
		})

		Convey("整数类型解析", func() {
			parser, _ := NewLineParserWithOptions[string, int](options)

			Convey("正常整数解析", func() {
				changeType, key, value, err := parser.Parse([]byte("count\t42"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "count")
				So(value, ShouldEqual, 42)
			})

			Convey("负数解析", func() {
				changeType, key, value, err := parser.Parse([]byte("temp\t-10"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "temp")
				So(value, ShouldEqual, -10)
			})

			Convey("无效整数", func() {
				changeType, key, value, err := parser.Parse([]byte("count\tabc"))
				So(err, ShouldNotBeNil)
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
				So(value, ShouldEqual, 0)
			})
		})

		Convey("浮点数类型解析", func() {
			parser, _ := NewLineParserWithOptions[string, float64](options)

			Convey("正常浮点数解析", func() {
				changeType, key, value, err := parser.Parse([]byte("pi\t3.14159"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "pi")
				So(value, ShouldEqual, 3.14159)
			})

			Convey("整数到浮点数", func() {
				changeType, key, value, err := parser.Parse([]byte("number\t42"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "number")
				So(value, ShouldEqual, 42.0)
			})
		})

		Convey("布尔类型解析", func() {
			parser, _ := NewLineParserWithOptions[string, bool](options)

			Convey("true值解析", func() {
				changeType, key, value, err := parser.Parse([]byte("enabled\ttrue"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "enabled")
				So(value, ShouldEqual, true)
			})

			Convey("false值解析", func() {
				changeType, key, value, err := parser.Parse([]byte("disabled\tfalse"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "disabled")
				So(value, ShouldEqual, false)
			})
		})

		Convey("复杂类型JSON解析", func() {
			type User struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}
			parser, _ := NewLineParserWithOptions[string, User](options)

			Convey("正常JSON解析", func() {
				changeType, key, value, err := parser.Parse([]byte("user1\t{\"name\":\"alice\",\"age\":25}"))
				So(err, ShouldBeNil)
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(key, ShouldEqual, "user1")
				So(value.Name, ShouldEqual, "alice")
				So(value.Age, ShouldEqual, 25)
			})

			Convey("无效JSON", func() {
				changeType, key, value, err := parser.Parse([]byte("user1\t{invalid json}"))
				So(err, ShouldNotBeNil)
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
				So(value.Name, ShouldEqual, "")
				So(value.Age, ShouldEqual, 0)
			})
		})

		Convey("自定义分隔符", func() {
			customOptions := &LineParserOptions{
				Separator: "|",
			}
			parser, _ := NewLineParserWithOptions[string, string](customOptions)

			changeType, key, value, err := parser.Parse([]byte("hello|world|update"))
			So(err, ShouldBeNil)
			So(changeType, ShouldEqual, ChangeTypeUpdate)
			So(key, ShouldEqual, "hello")
			So(value, ShouldEqual, "world")
		})
	})
}

func TestParseValue(t *testing.T) {
	Convey("parseValue", t, func() {
		Convey("字符串类型", func() {
			result, err := parseValue[string]("hello world")
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "hello world")
		})

		Convey("整数类型", func() {
			Convey("int类型", func() {
				result, err := parseValue[int]("42")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 42)
			})

			Convey("负数", func() {
				result, err := parseValue[int]("-10")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, -10)
			})

			Convey("int8类型", func() {
				result, err := parseValue[int8]("127")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int8(127))
			})

			Convey("int16类型", func() {
				result, err := parseValue[int16]("32767")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int16(32767))
			})

			Convey("int32类型", func() {
				result, err := parseValue[int32]("2147483647")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int32(2147483647))
			})

			Convey("int64类型", func() {
				result, err := parseValue[int64]("9223372036854775807")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(9223372036854775807))
			})

			Convey("无效整数", func() {
				result, err := parseValue[int]("abc")
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, 0)
			})
		})

		Convey("无符号整数类型", func() {
			Convey("uint类型", func() {
				result, err := parseValue[uint]("42")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, uint(42))
			})

			Convey("uint8类型", func() {
				result, err := parseValue[uint8]("255")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, uint8(255))
			})

			Convey("uint16类型", func() {
				result, err := parseValue[uint16]("65535")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, uint16(65535))
			})

			Convey("uint32类型", func() {
				result, err := parseValue[uint32]("4294967295")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, uint32(4294967295))
			})

			Convey("uint64类型", func() {
				result, err := parseValue[uint64]("18446744073709551615")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, uint64(18446744073709551615))
			})
		})

		Convey("浮点数类型", func() {
			Convey("float32类型", func() {
				result, err := parseValue[float32]("3.14")
				So(err, ShouldBeNil)
				So(result, ShouldAlmostEqual, float32(3.14), 0.001)
			})

			Convey("float64类型", func() {
				result, err := parseValue[float64]("3.141592653589793")
				So(err, ShouldBeNil)
				So(result, ShouldAlmostEqual, 3.141592653589793, 0.0000000000001)
			})

			Convey("科学计数法", func() {
				result, err := parseValue[float64]("1.23e-4")
				So(err, ShouldBeNil)
				So(result, ShouldAlmostEqual, 1.23e-4, 0.0000001)
			})

			Convey("无效浮点数", func() {
				result, err := parseValue[float64]("not a number")
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, 0.0)
			})
		})

		Convey("布尔类型", func() {
			Convey("true值", func() {
				testCases := []string{"true", "True", "TRUE", "1", "t", "T"}
				for _, testCase := range testCases {
					result, err := parseValue[bool](testCase)
					if err == nil && result {
						So(result, ShouldBeTrue)
					}
				}
			})

			Convey("false值", func() {
				testCases := []string{"false", "False", "FALSE", "0", "f", "F"}
				for _, testCase := range testCases {
					result, err := parseValue[bool](testCase)
					if err == nil && !result {
						So(result, ShouldBeFalse)
					}
				}
			})

			Convey("无效布尔值", func() {
				result, err := parseValue[bool]("maybe")
				So(err, ShouldNotBeNil)
				So(result, ShouldBeFalse)
			})
		})

		Convey("复杂类型JSON解析", func() {
			type Person struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}

			Convey("正常结构体解析", func() {
				result, err := parseValue[Person]("{\"name\":\"alice\",\"age\":25}")
				So(err, ShouldBeNil)
				So(result.Name, ShouldEqual, "alice")
				So(result.Age, ShouldEqual, 25)
			})

			Convey("切片解析", func() {
				result, err := parseValue[[]int]("[1,2,3,4,5]")
				So(err, ShouldBeNil)
				So(result, ShouldResemble, []int{1, 2, 3, 4, 5})
			})

			Convey("映射解析", func() {
				result, err := parseValue[map[string]int]("{\"a\":1,\"b\":2}")
				So(err, ShouldBeNil)
				So(result["a"], ShouldEqual, 1)
				So(result["b"], ShouldEqual, 2)
			})

			Convey("无效JSON", func() {
				result, err := parseValue[Person]("{invalid json}")
				So(err, ShouldNotBeNil)
				So(result.Name, ShouldEqual, "")
				So(result.Age, ShouldEqual, 0)
			})
		})

		Convey("边界情况", func() {
			Convey("空字符串", func() {
				Convey("字符串类型", func() {
					result, err := parseValue[string]("")
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "")
				})

				Convey("整数类型", func() {
					result, err := parseValue[int]("")
					So(err, ShouldNotBeNil)
					So(result, ShouldEqual, 0)
				})
			})

			Convey("空白字符", func() {
				Convey("字符串类型保留空白", func() {
					result, err := parseValue[string]("  \t\n  ")
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "  \t\n  ")
				})
			})
		})
	})
}
