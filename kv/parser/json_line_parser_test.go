package parser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewJsonLineParserWithOptions(t *testing.T) {
	Convey("NewJsonLineParserWithOptions", t, func() {
		Convey("创建基本JsonLineParser", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, err := NewJsonLineParserWithOptions[string, map[string]interface{}](options)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.keyFields, ShouldResemble, []string{"id"})
			So(parser.keySeparator, ShouldEqual, "_")
		})

		Convey("空配置使用默认值", func() {
			parser, err := NewJsonLineParserWithOptions[string, interface{}](nil)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.keySeparator, ShouldEqual, "_")
			So(len(parser.changeTypeRules), ShouldEqual, 0)
		})

		Convey("空分隔符使用默认值", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id"},
				KeySeparator: "",
			}
			parser, err := NewJsonLineParserWithOptions[string, interface{}](options)
			So(err, ShouldBeNil)
			So(parser.keySeparator, ShouldEqual, "_")
		})

		Convey("ChangeTypeRules配置", func() {
			options := &JsonLineParserOptions{
				KeyFields: []string{"id"},
				ChangeTypeRules: []ChangeTypeRule{
					{
						Logic: "and",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
						},
						Type: ChangeTypeDelete,
					},
					{
						Logic: "",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
						},
						Type: ChangeTypeUpdate,
					},
				},
			}
			parser, err := NewJsonLineParserWithOptions[string, interface{}](options)
			So(err, ShouldBeNil)
			So(len(parser.changeTypeRules), ShouldEqual, 2)
			So(parser.changeTypeRules[0].Logic, ShouldEqual, "AND")
			So(parser.changeTypeRules[1].Logic, ShouldEqual, "AND")
		})

		Convey("支持不同泛型类型", func() {
			options := &JsonLineParserOptions{
				KeyFields: []string{"id"},
			}

			Convey("int-string类型", func() {
				parser, err := NewJsonLineParserWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
			})

			Convey("string-User类型", func() {
				type User struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				}
				parser, err := NewJsonLineParserWithOptions[string, User](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
			})
		})
	})
}

func TestGetFieldValue(t *testing.T) {
	Convey("getFieldValue", t, func() {
		data := map[string]interface{}{
			"id":   "123",
			"name": "alice",
			"age":  25,
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"email":  "alice@example.com",
					"status": "active",
				},
				"settings": map[string]interface{}{
					"theme": "dark",
					"lang":  "en",
				},
			},
			"tags":   []string{"admin", "user"},
			"active": true,
		}

		Convey("简单字段提取", func() {
			Convey("字符串字段", func() {
				value, exists := getFieldValue(data, "name")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "alice")
			})

			Convey("数字字段", func() {
				value, exists := getFieldValue(data, "age")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, 25)
			})

			Convey("布尔字段", func() {
				value, exists := getFieldValue(data, "active")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, true)
			})

			Convey("数组字段", func() {
				value, exists := getFieldValue(data, "tags")
				So(exists, ShouldBeTrue)
				So(value, ShouldResemble, []string{"admin", "user"})
			})
		})

		Convey("嵌套字段提取", func() {
			Convey("二级嵌套", func() {
				value, exists := getFieldValue(data, "user.profile")
				So(exists, ShouldBeTrue)
				So(value, ShouldNotBeNil)
				_ = value
			})

			Convey("三级嵌套", func() {
				value, exists := getFieldValue(data, "user.profile.email")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "alice@example.com")
			})

			Convey("不同路径的嵌套", func() {
				value, exists := getFieldValue(data, "user.settings.theme")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "dark")
			})
		})

		Convey("错误情况", func() {
			Convey("字段不存在", func() {
				value, exists := getFieldValue(data, "nonexistent")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("嵌套字段不存在", func() {
				value, exists := getFieldValue(data, "user.nonexistent")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("路径中断（不是map类型）", func() {
				value, exists := getFieldValue(data, "name.profile")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("空字段路径", func() {
				value, exists := getFieldValue(data, "")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("数组字段继续访问", func() {
				value, exists := getFieldValue(data, "tags.length")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})
		})

		Convey("边界情况", func() {
			Convey("空数据", func() {
				emptyData := map[string]interface{}{}
				value, exists := getFieldValue(emptyData, "any")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("nil值字段", func() {
				dataWithNil := map[string]interface{}{
					"null_field": nil,
				}
				value, exists := getFieldValue(dataWithNil, "null_field")
				So(exists, ShouldBeTrue)
				So(value, ShouldBeNil)
			})
		})
	})
}

func TestGenerateKey(t *testing.T) {
	Convey("generateKey", t, func() {
		data := map[string]interface{}{
			"user_id": "123",
			"name":    "alice",
			"age":     25,
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"email": "alice@example.com",
				},
			},
		}

		Convey("单字段key生成", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			key, err := parser.generateKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123")
		})

		Convey("多字段key生成", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id", "name"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			key, err := parser.generateKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice")
		})

		Convey("嵌套字段key生成", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id", "user.profile.email"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			key, err := parser.generateKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice@example.com")
		})

		Convey("自定义分隔符", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id", "name", "age"},
				KeySeparator: "|",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			key, err := parser.generateKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123|alice|25")
		})

		Convey("数字类型字段", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"age", "user_id"},
				KeySeparator: "-",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			key, err := parser.generateKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "25-123")
		})

		Convey("错误情况", func() {
			Convey("无key字段配置", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "no key fields configured")
				So(key, ShouldEqual, "")
			})

			Convey("key字段不存在", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"nonexistent\" not found")
				So(key, ShouldEqual, "")
			})

			Convey("嵌套key字段不存在", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"user.nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"user.nonexistent\" not found")
				So(key, ShouldEqual, "")
			})

			Convey("部分字段存在", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"user_id", "nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"nonexistent\" not found")
				So(key, ShouldEqual, "")
			})
		})

		Convey("特殊值处理", func() {
			specialData := map[string]interface{}{
				"id":    "123",
				"zero":  0,
				"empty": "",
				"bool":  true,
				"null":  nil,
			}

			Convey("包含零值", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"id", "zero"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_0")
			})

			Convey("包含空字符串", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"id", "empty"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_")
			})

			Convey("包含布尔值", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"id", "bool"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_true")
			})

			Convey("包含nil值", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"id", "null"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				key, err := parser.generateKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_<nil>")
			})
		})
	})
}

func TestChangeTypeMatching(t *testing.T) {
	Convey("ChangeType匹配逻辑", t, func() {
		data := map[string]interface{}{
			"action": "delete",
			"status": "active",
			"user": map[string]interface{}{
				"role": "admin",
				"type": "premium",
			},
			"force": true,
		}

		Convey("compareValues函数", func() {
			Convey("相同类型比较", func() {
				So(compareValues("test", "test"), ShouldBeTrue)
				So(compareValues(123, 123), ShouldBeTrue)
				So(compareValues(true, true), ShouldBeTrue)
				So(compareValues(nil, nil), ShouldBeTrue)
			})

			Convey("不同类型比较", func() {
				So(compareValues("123", 123), ShouldBeTrue)
				So(compareValues(123, "123"), ShouldBeTrue)
				So(compareValues(true, "true"), ShouldBeTrue)
				So(compareValues("false", false), ShouldBeTrue)
			})

			Convey("不匹配的值", func() {
				So(compareValues("test", "other"), ShouldBeFalse)
				So(compareValues(123, 456), ShouldBeFalse)
				So(compareValues(nil, "test"), ShouldBeFalse)
				So(compareValues("test", nil), ShouldBeFalse)
			})
		})

		Convey("evaluateCondition函数", func() {
			Convey("简单字段条件", func() {
				condition := Condition{Field: "action", Value: "delete"}
				So(evaluateCondition(data, condition), ShouldBeTrue)

				condition = Condition{Field: "action", Value: "update"}
				So(evaluateCondition(data, condition), ShouldBeFalse)
			})

			Convey("嵌套字段条件", func() {
				condition := Condition{Field: "user.role", Value: "admin"}
				So(evaluateCondition(data, condition), ShouldBeTrue)

				condition = Condition{Field: "user.role", Value: "user"}
				So(evaluateCondition(data, condition), ShouldBeFalse)
			})

			Convey("字段不存在", func() {
				condition := Condition{Field: "nonexistent", Value: "any"}
				So(evaluateCondition(data, condition), ShouldBeFalse)
			})

			Convey("布尔值条件", func() {
				condition := Condition{Field: "force", Value: true}
				So(evaluateCondition(data, condition), ShouldBeTrue)

				condition = Condition{Field: "force", Value: false}
				So(evaluateCondition(data, condition), ShouldBeFalse)
			})
		})

		Convey("evaluateRule函数", func() {
			options := &JsonLineParserOptions{
				KeyFields: []string{"action"},
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			Convey("AND逻辑规则", func() {
				Convey("单条件匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})

				Convey("多条件都匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})

				Convey("部分条件不匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "inactive"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeFalse)
				})

				Convey("嵌套字段条件", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "user.role", Value: "admin"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})
			})

			Convey("OR逻辑规则", func() {
				Convey("第一条件匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "inactive"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})

				Convey("第二条件匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
							{Field: "status", Value: "active"},
						},
						Type: ChangeTypeUpdate,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})

				Convey("都匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})

				Convey("都不匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
							{Field: "status", Value: "inactive"},
						},
						Type: ChangeTypeUpdate,
					}
					So(parser.evaluateRule(data, rule), ShouldBeFalse)
				})
			})

			Convey("边界情况", func() {
				Convey("空条件列表", func() {
					rule := ChangeTypeRule{
						Logic:      "AND",
						Conditions: []Condition{},
						Type:       ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeFalse)
				})

				Convey("未知逻辑默认为AND", func() {
					rule := ChangeTypeRule{
						Logic: "UNKNOWN",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
						Type: ChangeTypeDelete,
					}
					So(parser.evaluateRule(data, rule), ShouldBeTrue)
				})
			})
		})

		Convey("determineChangeType函数", func() {
			Convey("第一个规则匹配", func() {
				options := &JsonLineParserOptions{
					KeyFields: []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "delete"},
							},
							Type: ChangeTypeDelete,
						},
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "status", Value: "active"},
							},
							Type: ChangeTypeUpdate,
						},
					},
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)
				So(parser.determineChangeType(data), ShouldEqual, ChangeTypeDelete)
			})

			Convey("第二个规则匹配", func() {
				options := &JsonLineParserOptions{
					KeyFields: []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "update"},
							},
							Type: ChangeTypeUpdate,
						},
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "delete"},
							},
							Type: ChangeTypeDelete,
						},
					},
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)
				So(parser.determineChangeType(data), ShouldEqual, ChangeTypeDelete)
			})

			Convey("无规则匹配使用默认值", func() {
				options := &JsonLineParserOptions{
					KeyFields: []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "update"},
							},
							Type: ChangeTypeUpdate,
						},
					},
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)
				So(parser.determineChangeType(data), ShouldEqual, ChangeTypeAdd)
			})

			Convey("空规则列表使用默认值", func() {
				options := &JsonLineParserOptions{
					KeyFields:       []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{},
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)
				So(parser.determineChangeType(data), ShouldEqual, ChangeTypeAdd)
			})

			Convey("复杂条件组合", func() {
				options := &JsonLineParserOptions{
					KeyFields: []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "delete"},
								{Field: "user.role", Value: "admin"},
								{Field: "force", Value: true},
							},
							Type: ChangeTypeDelete,
						},
					},
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)
				So(parser.determineChangeType(data), ShouldEqual, ChangeTypeDelete)
			})
		})
	})
}

func TestJsonLineParserParse(t *testing.T) {
	Convey("JsonLineParser.Parse", t, func() {
		Convey("基本JSON解析", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			jsonLine := `{"id":"123","name":"alice","age":25}`
			changeType, key, value, err := parser.Parse(jsonLine)

			So(err, ShouldBeNil)
			So(changeType, ShouldEqual, ChangeTypeAdd)
			So(key, ShouldEqual, "123")
			So(value, ShouldNotBeNil)

			// 验证value是完整的JSON对象
			valueMap := value.(map[string]interface{})
			So(valueMap["id"], ShouldEqual, "123")
			So(valueMap["name"], ShouldEqual, "alice")
			So(valueMap["age"], ShouldEqual, 25)
		})

		Convey("结构体类型解析", func() {
			type User struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Age  int    `json:"age"`
			}

			options := &JsonLineParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, User](options)

			jsonLine := `{"id":"123","name":"alice","age":25}`
			changeType, key, value, err := parser.Parse(jsonLine)

			So(err, ShouldBeNil)
			So(changeType, ShouldEqual, ChangeTypeAdd)
			So(key, ShouldEqual, "123")
			So(value.ID, ShouldEqual, "123")
			So(value.Name, ShouldEqual, "alice")
			So(value.Age, ShouldEqual, 25)
		})

		Convey("多字段key生成", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id", "name"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			jsonLine := `{"user_id":"123","name":"alice","action":"create"}`
			changeType, key, _, err := parser.Parse(jsonLine)

			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice")
			So(changeType, ShouldEqual, ChangeTypeAdd)
		})

		Convey("嵌套字段key生成", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user.id", "user.profile.email"},
				KeySeparator: "|",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			jsonLine := `{"user":{"id":"123","profile":{"email":"alice@test.com"}},"action":"update"}`
			changeType, key, _, err := parser.Parse(jsonLine)

			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123|alice@test.com")
			So(changeType, ShouldEqual, ChangeTypeAdd)
		})

		Convey("ChangeType规则匹配", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
				ChangeTypeRules: []ChangeTypeRule{
					{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
						},
						Type: ChangeTypeDelete,
					},
					{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
							{Field: "status", Value: "active"},
						},
						Type: ChangeTypeUpdate,
					},
				},
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			Convey("匹配delete规则", func() {
				jsonLine := `{"id":"123","action":"delete","status":"any"}`
				changeType, key, value, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeDelete)
				So(value, ShouldNotBeNil)
			})

			Convey("匹配update规则", func() {
				jsonLine := `{"id":"456","action":"update","status":"active"}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "456")
				So(changeType, ShouldEqual, ChangeTypeUpdate)
			})

			Convey("无规则匹配使用默认", func() {
				jsonLine := `{"id":"789","action":"create","status":"pending"}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "789")
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})
		})

		Convey("复杂条件规则", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"user_id"},
				KeySeparator: "_",
				ChangeTypeRules: []ChangeTypeRule{
					{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "remove"},
							{Field: "action", Value: "delete"},
						},
						Type: ChangeTypeDelete,
					},
					{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "user.role", Value: "admin"},
							{Field: "force", Value: true},
						},
						Type: ChangeTypeUpdate,
					},
				},
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			Convey("OR逻辑匹配", func() {
				jsonLine := `{"user_id":"123","action":"remove","user":{"role":"user"}}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeDelete)
			})

			Convey("AND逻辑匹配", func() {
				jsonLine := `{"user_id":"456","action":"modify","user":{"role":"admin"},"force":true}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "456")
				So(changeType, ShouldEqual, ChangeTypeUpdate)
			})
		})

		Convey("错误情况", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			Convey("无效JSON", func() {
				jsonLine := `{"id":"123","name":}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to parse JSON")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
			})

			Convey("key字段缺失", func() {
				jsonLine := `{"name":"alice","age":25}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to generate key")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
			})

			Convey("key类型转换失败", func() {
				jsonLine := `{"id":"not_a_number"}`
				parser, _ := NewJsonLineParserWithOptions[int, interface{}](options)

				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to convert key")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, 0)
			})

			Convey("value类型转换失败", func() {
				type InvalidStruct struct {
					ID   int `json:"id"`
					Data int `json:"data"`
				}

				jsonLine := `{"id":"123","data":"not_a_number"}`
				parser, _ := NewJsonLineParserWithOptions[string, InvalidStruct](options)

				changeType, _, _, err := parser.Parse(jsonLine)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to unmarshal JSON")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
			})
		})

		Convey("特殊值处理", func() {
			options := &JsonLineParserOptions{
				KeyFields:    []string{"id", "status"},
				KeySeparator: "_",
			}
			parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

			Convey("包含null值", func() {
				jsonLine := `{"id":"123","status":null,"data":"test"}`
				changeType, key, value, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_<nil>")
				So(changeType, ShouldEqual, ChangeTypeAdd)

				valueMap := value.(map[string]interface{})
				So(valueMap["status"], ShouldBeNil)
			})

			Convey("包含嵌套对象", func() {
				jsonLine := `{"id":"123","status":"active","user":{"name":"alice","settings":{"theme":"dark"}}}`
				changeType, key, value, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_active")
				So(changeType, ShouldEqual, ChangeTypeAdd)

				valueMap := value.(map[string]interface{})
				userMap := valueMap["user"].(map[string]interface{})
				So(userMap["name"], ShouldEqual, "alice")
			})

			Convey("包含数组", func() {
				jsonLine := `{"id":"123","status":"active","tags":["admin","user"]}`
				changeType, key, value, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_active")
				So(changeType, ShouldEqual, ChangeTypeAdd)

				valueMap := value.(map[string]interface{})
				tags := valueMap["tags"].([]interface{})
				So(len(tags), ShouldEqual, 2)
				So(tags[0], ShouldEqual, "admin")
				So(tags[1], ShouldEqual, "user")
			})
		})

		Convey("不同key类型", func() {
			Convey("int类型key", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"id"},
					KeySeparator: "_",
				}
				parser, _ := NewJsonLineParserWithOptions[int, interface{}](options)

				jsonLine := `{"id":"123","name":"alice"}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, 123)
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})

			Convey("复合key类型", func() {
				options := &JsonLineParserOptions{
					KeyFields:    []string{"user_id", "timestamp"},
					KeySeparator: "-",
				}
				parser, _ := NewJsonLineParserWithOptions[string, interface{}](options)

				jsonLine := `{"user_id":"alice","timestamp":1609459200,"action":"login"}`
				changeType, key, _, err := parser.Parse(jsonLine)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "alice-1609459200")
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})
		})
	})
}