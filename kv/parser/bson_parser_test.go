package parser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
)

func TestNewBsonParserWithOptions(t *testing.T) {
	Convey("NewBsonParserWithOptions", t, func() {
		Convey("创建基本BsonParser", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, err := NewBsonParserWithOptions[string, bson.M](options)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.keyFields, ShouldResemble, []string{"id"})
			So(parser.keySeparator, ShouldEqual, "_")
		})

		Convey("空配置使用默认值", func() {
			parser, err := NewBsonParserWithOptions[string, interface{}](nil)
			So(err, ShouldBeNil)
			So(parser, ShouldNotBeNil)
			So(parser.keySeparator, ShouldEqual, "_")
			So(parser.keyFields, ShouldResemble, []string{"id"})
			So(len(parser.changeTypeRules), ShouldEqual, 0)
		})

		Convey("空分隔符使用默认值", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id"},
				KeySeparator: "",
			}
			parser, err := NewBsonParserWithOptions[string, interface{}](options)
			So(err, ShouldBeNil)
			So(parser.keySeparator, ShouldEqual, "_")
		})

		Convey("ChangeTypeRules配置", func() {
			options := &BsonParserOptions{
				KeyFields: []string{"id"},
				ChangeTypeRules: []ChangeTypeRule{
					{
						Logic: "AND",
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
			parser, err := NewBsonParserWithOptions[string, interface{}](options)
			So(err, ShouldBeNil)
			So(len(parser.changeTypeRules), ShouldEqual, 2)
			So(parser.changeTypeRules[0].Logic, ShouldEqual, "AND")
			So(parser.changeTypeRules[1].Logic, ShouldEqual, "AND")
		})

		Convey("支持不同泛型类型", func() {
			options := &BsonParserOptions{
				KeyFields: []string{"id"},
			}

			Convey("int-string类型", func() {
				parser, err := NewBsonParserWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
			})

			Convey("string-User类型", func() {
				type User struct {
					Name string `bson:"name"`
					Age  int    `bson:"age"`
				}
				parser, err := NewBsonParserWithOptions[string, User](options)
				So(err, ShouldBeNil)
				So(parser, ShouldNotBeNil)
			})
		})
	})
}

func TestGetBsonFieldValue(t *testing.T) {
	Convey("getBsonFieldValue", t, func() {
		data := bson.M{
			"user_id": "123",
			"name":    "alice",
			"age":     25,
			"active":  true,
			"tags":    []string{"admin", "user"},
			"user": bson.M{
				"profile": bson.M{
					"email": "alice@example.com",
					"phone": "1234567890",
				},
				"role": "admin",
			},
		}

		Convey("简单字段提取", func() {
			Convey("字符串字段", func() {
				value, exists := getBsonFieldValue(data, "name")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "alice")
			})

			Convey("数字字段", func() {
				value, exists := getBsonFieldValue(data, "age")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, 25)
			})

			Convey("布尔字段", func() {
				value, exists := getBsonFieldValue(data, "active")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, true)
			})

			Convey("数组字段", func() {
				value, exists := getBsonFieldValue(data, "tags")
				So(exists, ShouldBeTrue)
				So(value, ShouldResemble, []string{"admin", "user"})
			})
		})

		Convey("嵌套字段提取", func() {
			Convey("二级嵌套", func() {
				value, exists := getBsonFieldValue(data, "user.role")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "admin")
			})

			Convey("三级嵌套", func() {
				value, exists := getBsonFieldValue(data, "user.profile.email")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "alice@example.com")
			})

			Convey("不同路径的嵌套", func() {
				value, exists := getBsonFieldValue(data, "user.profile.phone")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "1234567890")
			})
		})

		Convey("错误情况", func() {
			Convey("字段不存在", func() {
				value, exists := getBsonFieldValue(data, "nonexistent")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("嵌套字段不存在", func() {
				value, exists := getBsonFieldValue(data, "user.nonexistent")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("路径中断（不是bson.M类型）", func() {
				value, exists := getBsonFieldValue(data, "age.invalid")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("空字段路径", func() {
				value, exists := getBsonFieldValue(data, "")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("数组字段继续访问", func() {
				value, exists := getBsonFieldValue(data, "tags.0")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})
		})

		Convey("边界情况", func() {
			Convey("空数据", func() {
				emptyData := bson.M{}
				value, exists := getBsonFieldValue(emptyData, "name")
				So(exists, ShouldBeFalse)
				So(value, ShouldBeNil)
			})

			Convey("nil值字段", func() {
				nilData := bson.M{"null_field": nil}
				value, exists := getBsonFieldValue(nilData, "null_field")
				So(exists, ShouldBeTrue)
				So(value, ShouldBeNil)
			})
		})
	})
}

func TestGenerateBsonKey(t *testing.T) {
	Convey("generateBsonKey", t, func() {
		data := bson.M{
			"user_id": "123",
			"name":    "alice",
			"age":     25,
			"user": bson.M{
				"profile": bson.M{
					"email": "alice@example.com",
				},
			},
		}

		Convey("单字段key生成", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			key, err := parser.generateBsonKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123")
		})

		Convey("多字段key生成", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id", "name"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			key, err := parser.generateBsonKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice")
		})

		Convey("嵌套字段key生成", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id", "user.profile.email"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			key, err := parser.generateBsonKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice@example.com")
		})

		Convey("自定义分隔符", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id", "name", "age"},
				KeySeparator: "|",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			key, err := parser.generateBsonKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123|alice|25")
		})

		Convey("数字类型字段", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"age", "user_id"},
				KeySeparator: "-",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			key, err := parser.generateBsonKey(data)
			So(err, ShouldBeNil)
			So(key, ShouldEqual, "25-123")
		})

		Convey("错误情况", func() {
			Convey("无key字段配置", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "no key fields configured")
				So(key, ShouldEqual, "")
			})

			Convey("key字段不存在", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"nonexistent\" not found")
				So(key, ShouldEqual, "")
			})

			Convey("嵌套key字段不存在", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"user.nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"user.nonexistent\" not found")
				So(key, ShouldEqual, "")
			})

			Convey("部分字段存在", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"user_id", "nonexistent"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key field \"nonexistent\" not found")
				So(key, ShouldEqual, "")
			})
		})

		Convey("特殊值处理", func() {
			specialData := bson.M{
				"id":    "123",
				"zero":  0,
				"empty": "",
				"bool":  true,
				"null":  nil,
			}

			Convey("包含零值", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"id", "zero"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_0")
			})

			Convey("包含空字符串", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"id", "empty"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_")
			})

			Convey("包含布尔值", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"id", "bool"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_true")
			})

			Convey("包含nil值", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"id", "null"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				key, err := parser.generateBsonKey(specialData)
				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_<nil>")
			})
		})
	})
}

func TestBsonChangeTypeMatching(t *testing.T) {
	Convey("BSON ChangeType匹配逻辑", t, func() {
		Convey("evaluateBsonCondition函数", func() {
			data := bson.M{
				"action": "delete",
				"status": "active",
				"user": bson.M{
					"role":  "admin",
					"force": true,
				},
			}

			Convey("简单字段条件", func() {
				condition := Condition{Field: "action", Value: "delete"}
				So(evaluateBsonCondition(data, condition), ShouldBeTrue)
			})

			Convey("嵌套字段条件", func() {
				condition := Condition{Field: "user.role", Value: "admin"}
				So(evaluateBsonCondition(data, condition), ShouldBeTrue)
			})

			Convey("字段不存在", func() {
				condition := Condition{Field: "nonexistent", Value: "any"}
				So(evaluateBsonCondition(data, condition), ShouldBeFalse)
			})

			Convey("布尔值条件", func() {
				condition := Condition{Field: "user.force", Value: true}
				So(evaluateBsonCondition(data, condition), ShouldBeTrue)
			})
		})

		Convey("evaluateBsonRule函数", func() {
			options := &BsonParserOptions{
				KeyFields: []string{"action"},
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			data := bson.M{
				"action": "delete",
				"status": "active",
				"user": bson.M{
					"role":  "admin",
					"force": true,
				},
			}

			Convey("AND逻辑规则", func() {
				Convey("单条件匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})

				Convey("多条件都匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})

				Convey("部分条件不匹配", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "inactive"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeFalse)
				})

				Convey("嵌套字段条件", func() {
					rule := ChangeTypeRule{
						Logic: "AND",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "user.role", Value: "admin"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
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
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})

				Convey("第二条件匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
							{Field: "status", Value: "active"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})

				Convey("都匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})

				Convey("都不匹配", func() {
					rule := ChangeTypeRule{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "update"},
							{Field: "status", Value: "inactive"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeFalse)
				})
			})

			Convey("边界情况", func() {
				Convey("空条件列表", func() {
					rule := ChangeTypeRule{
						Logic:      "AND",
						Conditions: []Condition{},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeFalse)
				})

				Convey("未知逻辑默认为AND", func() {
					rule := ChangeTypeRule{
						Logic: "UNKNOWN",
						Conditions: []Condition{
							{Field: "action", Value: "delete"},
							{Field: "status", Value: "active"},
						},
					}
					So(parser.evaluateBsonRule(data, rule), ShouldBeTrue)
				})
			})
		})

		Convey("determineBsonChangeType函数", func() {
			data := bson.M{
				"action": "delete",
				"status": "active",
				"user": bson.M{
					"role":  "admin",
					"force": true,
				},
			}

			Convey("第一个规则匹配", func() {
				options := &BsonParserOptions{
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
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)
				So(parser.determineBsonChangeType(data), ShouldEqual, ChangeTypeDelete)
			})

			Convey("第二个规则匹配", func() {
				options := &BsonParserOptions{
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
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)
				So(parser.determineBsonChangeType(data), ShouldEqual, ChangeTypeDelete)
			})

			Convey("无规则匹配使用默认值", func() {
				options := &BsonParserOptions{
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
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)
				So(parser.determineBsonChangeType(data), ShouldEqual, ChangeTypeAdd)
			})

			Convey("空规则列表使用默认值", func() {
				options := &BsonParserOptions{
					KeyFields:       []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{},
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)
				So(parser.determineBsonChangeType(data), ShouldEqual, ChangeTypeAdd)
			})

			Convey("复杂条件组合", func() {
				options := &BsonParserOptions{
					KeyFields: []string{"action"},
					ChangeTypeRules: []ChangeTypeRule{
						{
							Logic: "AND",
							Conditions: []Condition{
								{Field: "action", Value: "delete"},
								{Field: "user.role", Value: "admin"},
								{Field: "user.force", Value: true},
							},
							Type: ChangeTypeDelete,
						},
					},
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)
				So(parser.determineBsonChangeType(data), ShouldEqual, ChangeTypeDelete)
			})
		})
	})
}

func TestBsonParserParse(t *testing.T) {
	Convey("BsonParser.Parse", t, func() {
		Convey("基本BSON解析", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			data := bson.M{"id": "123", "name": "alice", "age": 25}
			bsonData, _ := bson.Marshal(data)

			changeType, key, value, err := parser.Parse(bsonData)

			So(err, ShouldBeNil)
			So(changeType, ShouldEqual, ChangeTypeAdd)
			So(key, ShouldEqual, "123")
			So(value, ShouldNotBeNil)

			// 验证value是完整的BSON对象
			valueMap := value.(bson.M)
			So(valueMap["id"], ShouldEqual, "123")
			So(valueMap["name"], ShouldEqual, "alice")
			So(valueMap["age"], ShouldEqual, int32(25))
		})

		Convey("结构体类型解析", func() {
			type User struct {
				ID   string `bson:"id"`
				Name string `bson:"name"`
				Age  int    `bson:"age"`
			}

			options := &BsonParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, User](options)

			userData := User{ID: "123", Name: "alice", Age: 25}
			bsonData, _ := bson.Marshal(userData)

			changeType, key, value, err := parser.Parse(bsonData)

			So(err, ShouldBeNil)
			So(changeType, ShouldEqual, ChangeTypeAdd)
			So(key, ShouldEqual, "123")
			So(value.ID, ShouldEqual, "123")
			So(value.Name, ShouldEqual, "alice")
			So(value.Age, ShouldEqual, 25)
		})

		Convey("多字段key生成", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user_id", "name"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			data := bson.M{"user_id": "123", "name": "alice", "action": "create"}
			bsonData, _ := bson.Marshal(data)

			changeType, key, _, err := parser.Parse(bsonData)

			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123_alice")
			So(changeType, ShouldEqual, ChangeTypeAdd)
		})

		Convey("嵌套字段key生成", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"user.id", "user.profile.email"},
				KeySeparator: "|",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			data := bson.M{
				"user": bson.M{
					"id": "123",
					"profile": bson.M{
						"email": "alice@test.com",
					},
				},
				"action": "update",
			}
			bsonData, _ := bson.Marshal(data)

			changeType, key, _, err := parser.Parse(bsonData)

			So(err, ShouldBeNil)
			So(key, ShouldEqual, "123|alice@test.com")
			So(changeType, ShouldEqual, ChangeTypeAdd)
		})

		Convey("ChangeType规则匹配", func() {
			options := &BsonParserOptions{
				KeyFields: []string{"id"},
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
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			Convey("匹配delete规则", func() {
				data := bson.M{"id": "123", "action": "delete", "status": "any"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, value, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeDelete)
				So(value, ShouldNotBeNil)
			})

			Convey("匹配update规则", func() {
				data := bson.M{"id": "123", "action": "update", "status": "active"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeUpdate)
			})

			Convey("无规则匹配使用默认", func() {
				data := bson.M{"id": "123", "action": "create", "status": "pending"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})
		})

		Convey("复杂条件规则", func() {
			options := &BsonParserOptions{
				KeyFields: []string{"user_id"},
				ChangeTypeRules: []ChangeTypeRule{
					{
						Logic: "OR",
						Conditions: []Condition{
							{Field: "action", Value: "remove"},
							{Field: "user.role", Value: "admin"},
							{Field: "force", Value: true},
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
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			Convey("OR逻辑匹配", func() {
				data := bson.M{
					"user_id": "123",
					"action":  "remove",
					"user": bson.M{
						"role": "user",
					},
				}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123")
				So(changeType, ShouldEqual, ChangeTypeDelete)
			})

			Convey("AND逻辑匹配", func() {
				data := bson.M{
					"user_id": "456",
					"action":  "modify",
					"user": bson.M{
						"role": "admin",
					},
					"force": true,
				}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "456")
				// 由于第一个规则是OR逻辑，user.role=admin 也满足第一个规则，所以返回 ChangeTypeDelete
				So(changeType, ShouldEqual, ChangeTypeDelete)
			})
		})

		Convey("错误情况", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"id"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			Convey("无效BSON", func() {
				invalidBson := []byte{0x00, 0x01, 0x02}

				changeType, key, _, err := parser.Parse(invalidBson)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to parse BSON")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
			})

			Convey("key字段缺失", func() {
				data := bson.M{"name": "alice", "age": 25}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to generate key")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, "")
			})

			Convey("key类型转换失败", func() {
				data := bson.M{"id": "not_a_number"}
				bsonData, _ := bson.Marshal(data)
				parser, _ := NewBsonParserWithOptions[int, interface{}](options)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to convert key")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
				So(key, ShouldEqual, 0)
			})

			Convey("value类型转换失败", func() {
				type InvalidStruct struct {
					ID   int `bson:"id"`
					Data int `bson:"data"`
				}

				data := bson.M{"id": "123", "data": "not_a_number"}
				bsonData, _ := bson.Marshal(data)
				parser, _ := NewBsonParserWithOptions[string, InvalidStruct](options)

				changeType, _, _, err := parser.Parse(bsonData)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to unmarshal BSON")
				So(changeType, ShouldEqual, ChangeTypeUnknown)
			})
		})

		Convey("特殊值处理", func() {
			options := &BsonParserOptions{
				KeyFields:    []string{"id", "status"},
				KeySeparator: "_",
			}
			parser, _ := NewBsonParserWithOptions[string, interface{}](options)

			Convey("包含null值", func() {
				data := bson.M{"id": "123", "status": nil, "data": "test"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, value, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_<nil>")
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(value, ShouldNotBeNil)
			})

			Convey("包含嵌套对象", func() {
				data := bson.M{
					"id":     "123",
					"status": "active",
					"user": bson.M{
						"name":  "alice",
						"email": "alice@test.com",
					},
				}
				bsonData, _ := bson.Marshal(data)

				changeType, key, value, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_active")
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(value, ShouldNotBeNil)

				valueMap := value.(bson.M)
				userMap := valueMap["user"].(bson.M)
				So(userMap["name"], ShouldEqual, "alice")
				So(userMap["email"], ShouldEqual, "alice@test.com")
			})

			Convey("包含数组", func() {
				data := bson.M{
					"id":     "123",
					"status": "active",
					"tags":   []string{"admin", "user", "test"},
					"scores": []int{95, 87, 92},
				}
				bsonData, _ := bson.Marshal(data)

				changeType, key, value, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "123_active")
				So(changeType, ShouldEqual, ChangeTypeAdd)
				So(value, ShouldNotBeNil)

				valueMap := value.(bson.M)
				So(len(valueMap["tags"].(bson.A)), ShouldEqual, 3)
				So(len(valueMap["scores"].(bson.A)), ShouldEqual, 3)
			})
		})

		Convey("不同key类型", func() {
			Convey("int类型key", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"id"},
					KeySeparator: "_",
				}
				parser, _ := NewBsonParserWithOptions[int, interface{}](options)

				data := bson.M{"id": "123", "name": "alice"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, 123)
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})

			Convey("复合key类型", func() {
				options := &BsonParserOptions{
					KeyFields:    []string{"user_id", "timestamp"},
					KeySeparator: "-",
				}
				parser, _ := NewBsonParserWithOptions[string, interface{}](options)

				data := bson.M{"user_id": "alice", "timestamp": 1609459200, "action": "login"}
				bsonData, _ := bson.Marshal(data)

				changeType, key, _, err := parser.Parse(bsonData)

				So(err, ShouldBeNil)
				So(key, ShouldEqual, "alice-1609459200")
				So(changeType, ShouldEqual, ChangeTypeAdd)
			})
		})
	})
}