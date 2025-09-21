package parser

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

type BsonParserOptions struct {
	KeyFields       []string         `cfg:"keyFields"`            // 用于生成key的字段路径
	KeySeparator    string           `cfg:"keySeparator" def:"_"` // key字段间的分隔符
	ChangeTypeRules []ChangeTypeRule `cfg:"changeTypeRules"`      // changeType规则
}

type BsonParser[K, V any] struct {
	keyFields       []string
	keySeparator    string
	changeTypeRules []ChangeTypeRule
}

func NewBsonParserWithOptions[K, V any](options *BsonParserOptions) (*BsonParser[K, V], error) {
	if options == nil {
		return &BsonParser[K, V]{
			keyFields:    []string{"id"},
			keySeparator: "_",
		}, nil
	}

	keySeparator := options.KeySeparator
	if keySeparator == "" {
		keySeparator = "_"
	}

	// 为每个规则设置默认Logic
	rules := make([]ChangeTypeRule, len(options.ChangeTypeRules))
	for i, rule := range options.ChangeTypeRules {
		rules[i] = rule
		if rules[i].Logic == "" {
			rules[i].Logic = "AND"
		}
		rules[i].Logic = strings.ToUpper(rules[i].Logic)
	}

	return &BsonParser[K, V]{
		keyFields:       options.KeyFields,
		keySeparator:    keySeparator,
		changeTypeRules: rules,
	}, nil
}

// getBsonFieldValue 从BSON对象中提取指定路径的字段值
func getBsonFieldValue(data bson.M, fieldPath string) (interface{}, bool) {
	if fieldPath == "" {
		return nil, false
	}

	// 支持嵌套路径，如 "user.id" 或 "metadata.timestamp"
	parts := strings.Split(fieldPath, ".")
	current := data

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, false
		}

		// 如果是最后一个部分，返回值
		if i == len(parts)-1 {
			return value, true
		}

		// 否则继续向下遍历，检查是否是bson.M类型
		if nextMap, ok := value.(bson.M); ok {
			current = nextMap
		} else {
			// 不是bson.M类型，无法继续遍历
			return nil, false
		}
	}

	return nil, false
}

// generateBsonKey 根据配置的字段生成key
func (p *BsonParser[K, V]) generateBsonKey(data bson.M) (string, error) {
	if len(p.keyFields) == 0 {
		return "", fmt.Errorf("no key fields configured")
	}

	var keyParts []string
	for _, field := range p.keyFields {
		value, exists := getBsonFieldValue(data, field)
		if !exists {
			return "", fmt.Errorf("key field %q not found in BSON", field)
		}

		// 将值转换为字符串
		keyParts = append(keyParts, fmt.Sprintf("%v", value))
	}

	return strings.Join(keyParts, p.keySeparator), nil
}

// evaluateBsonCondition 评估单个条件是否满足
func evaluateBsonCondition(data bson.M, condition Condition) bool {
	value, exists := getBsonFieldValue(data, condition.Field)
	if !exists {
		return false
	}

	return compareValues(value, condition.Value)
}

// evaluateBsonRule 评估规则是否匹配
func (p *BsonParser[K, V]) evaluateBsonRule(data bson.M, rule ChangeTypeRule) bool {
	if len(rule.Conditions) == 0 {
		return false
	}

	switch rule.Logic {
	case "AND":
		// 所有条件都必须满足
		for _, condition := range rule.Conditions {
			if !evaluateBsonCondition(data, condition) {
				return false
			}
		}
		return true

	case "OR":
		// 任一条件满足即可
		for _, condition := range rule.Conditions {
			if evaluateBsonCondition(data, condition) {
				return true
			}
		}
		return false

	default:
		// 默认使用AND逻辑
		for _, condition := range rule.Conditions {
			if !evaluateBsonCondition(data, condition) {
				return false
			}
		}
		return true
	}
}

// determineBsonChangeType 根据规则确定changeType
func (p *BsonParser[K, V]) determineBsonChangeType(data bson.M) ChangeType {
	// 按顺序检查规则，第一个匹配的规则决定changeType
	for _, rule := range p.changeTypeRules {
		if p.evaluateBsonRule(data, rule) {
			return rule.Type
		}
	}

	// 都不匹配时使用默认值
	return ChangeTypeAdd
}

func (p *BsonParser[K, V]) Parse(buf []byte) (ChangeType, K, V, error) {
	var zeroK K
	var zeroV V

	// 解析BSON
	var bsonData bson.M
	if err := bson.Unmarshal(buf, &bsonData); err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to parse BSON: %w", err)
	}

	// 生成key
	keyStr, err := p.generateBsonKey(bsonData)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to generate key: %w", err)
	}

	// 转换key到目标类型
	key, err := parseValue[K](keyStr)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to convert key to type %T: %w", zeroK, err)
	}

	// 将整个BSON作为value
	var value V

	// 检查V的类型
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		// V是interface{}类型，直接使用bsonData
		if v, ok := any(bsonData).(V); ok {
			value = v
		} else {
			return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to convert BSON data to type %T", zeroV)
		}
	} else {
		// V是具体类型，使用BSON反序列化
		if err := bson.Unmarshal(buf, &value); err != nil {
			return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to unmarshal BSON to type %T: %w", zeroV, err)
		}
	}

	// 确定changeType
	changeType := p.determineBsonChangeType(bsonData)

	return changeType, key, value, nil
}