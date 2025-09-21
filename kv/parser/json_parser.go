package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Condition struct {
	Field string      `cfg:"field"` // 字段路径，支持嵌套 "user.status"
	Value interface{} `cfg:"value"` // 期望值
}

type ChangeTypeRule struct {
	Conditions []Condition `cfg:"conditions"`      // 条件列表
	Logic      string      `cfg:"logic" def:"AND"` // 逻辑关系: "AND" 或 "OR"
	Type       ChangeType  `cfg:"type"`            // 满足条件时的changeType
}

type JsonLineParserOptions struct {
	KeyFields       []string        `cfg:"keyFields"`                 // 用于生成key的字段路径
	KeySeparator    string          `cfg:"keySeparator" def:"_"`      // key字段间的分隔符
	ChangeTypeRules []ChangeTypeRule `cfg:"changeTypeRules"`          // changeType规则
}

type JsonLineParser[K, V any] struct {
	keyFields       []string
	keySeparator    string
	changeTypeRules []ChangeTypeRule
}

func NewJsonLineParserWithOptions[K, V any](options *JsonLineParserOptions) (*JsonLineParser[K, V], error) {
	if options == nil {
		options = &JsonLineParserOptions{}
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
	
	return &JsonLineParser[K, V]{
		keyFields:       options.KeyFields,
		keySeparator:    keySeparator,
		changeTypeRules: rules,
	}, nil
}

// getFieldValue 从JSON对象中提取指定路径的字段值
func getFieldValue(data map[string]interface{}, fieldPath string) (interface{}, bool) {
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
		
		// 否则继续向下遍历，检查是否是map类型
		if nextMap, ok := value.(map[string]interface{}); ok {
			current = nextMap
		} else {
			// 不是map类型，无法继续遍历
			return nil, false
		}
	}
	
	return nil, false
}

// generateKey 根据配置的字段生成key
func (p *JsonLineParser[K, V]) generateKey(data map[string]interface{}) (string, error) {
	if len(p.keyFields) == 0 {
		return "", fmt.Errorf("no key fields configured")
	}
	
	var keyParts []string
	for _, field := range p.keyFields {
		value, exists := getFieldValue(data, field)
		if !exists {
			return "", fmt.Errorf("key field %q not found in JSON", field)
		}
		
		// 将值转换为字符串
		keyParts = append(keyParts, fmt.Sprintf("%v", value))
	}
	
	return strings.Join(keyParts, p.keySeparator), nil
}

// compareValues 比较两个值是否相等，支持类型转换
func compareValues(actual, expected interface{}) bool {
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}
	
	// 直接比较
	if actual == expected {
		return true
	}
	
	// 类型转换比较
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return actualStr == expectedStr
}

// evaluateCondition 评估单个条件是否满足
func evaluateCondition(data map[string]interface{}, condition Condition) bool {
	value, exists := getFieldValue(data, condition.Field)
	if !exists {
		return false
	}
	
	return compareValues(value, condition.Value)
}

// evaluateRule 评估规则是否匹配
func (p *JsonLineParser[K, V]) evaluateRule(data map[string]interface{}, rule ChangeTypeRule) bool {
	if len(rule.Conditions) == 0 {
		return false
	}
	
	switch rule.Logic {
	case "AND":
		// 所有条件都必须满足
		for _, condition := range rule.Conditions {
			if !evaluateCondition(data, condition) {
				return false
			}
		}
		return true
		
	case "OR":
		// 任一条件满足即可
		for _, condition := range rule.Conditions {
			if evaluateCondition(data, condition) {
				return true
			}
		}
		return false
		
	default:
		// 默认使用AND逻辑
		for _, condition := range rule.Conditions {
			if !evaluateCondition(data, condition) {
				return false
			}
		}
		return true
	}
}

// determineChangeType 根据规则确定changeType
func (p *JsonLineParser[K, V]) determineChangeType(data map[string]interface{}) ChangeType {
	// 按顺序检查规则，第一个匹配的规则决定changeType
	for _, rule := range p.changeTypeRules {
		if p.evaluateRule(data, rule) {
			return rule.Type
		}
	}
	
	// 都不匹配时使用默认值
	return ChangeTypeAdd
}

func (p *JsonLineParser[K, V]) Parse(line string) (ChangeType, K, V, error) {
	var zeroK K
	var zeroV V
	
	// 解析JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// 生成key
	keyStr, err := p.generateKey(jsonData)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to generate key: %w", err)
	}
	
	// 转换key到目标类型
	key, err := parseValue[K](keyStr)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to convert key to type %T: %w", zeroK, err)
	}
	
	// 将整个JSON作为value
	var value V
	
	// 检查V的类型
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		// V是interface{}类型，直接使用jsonData
		if v, ok := any(jsonData).(V); ok {
			value = v
		} else {
			return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to convert JSON data to type %T", zeroV)
		}
	} else {
		// V是具体类型，使用JSON反序列化
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to unmarshal JSON to type %T: %w", zeroV, err)
		}
	}
	
	// 确定changeType
	changeType := p.determineChangeType(jsonData)
	
	return changeType, key, value, nil
}
