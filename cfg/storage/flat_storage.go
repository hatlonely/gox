package storage

import "strings"

type FlatStorage struct {
	data           map[string]interface{}
	separator      string
	enableDefaults bool

	parent *FlatStorage
	prefix string
}

func NewFlatStorage(data map[string]interface{}) *FlatStorage {
	return &FlatStorage{
		data:      data,
		separator: ".",
	}
}

func (fs *FlatStorage) WithDefaults(enable bool) *FlatStorage {
	fs.enableDefaults = enable
	return fs
}

func (fs *FlatStorage) WithSeparator(sep string) *FlatStorage {
	fs.separator = sep
	return fs
}

func (fs *FlatStorage) Data() map[string]interface{} {
	return fs.data
}

func (fs *FlatStorage) Sub(key string) Storage {
	if key == "" {
		return fs
	}

	if fs.parent != nil {
		return fs.parent.Sub(fs.prefix + "." + key)
	}

	keys := fs.parseKey(key)

	return &FlatStorage{
		parent: fs,
		prefix: strings.Join(keys, fs.separator),
	}
}

func (fs *FlatStorage) ConvertTo(object interface{}) error {
	return nil
}

func (fs *FlatStorage) Equals(other Storage) bool {
	return false
}

// parseKey 解析 key 字符串，支持点号和数组索引
func (ms *FlatStorage) parseKey(key string) []string {
	var keys []string
	var current string
	inBracket := false

	for _, char := range key {
		switch char {
		case '.':
			if !inBracket {
				if current != "" {
					keys = append(keys, current)
					current = ""
				}
			} else {
				current += string(char)
			}
		case '[':
			if current != "" {
				keys = append(keys, current)
				current = ""
			}
			inBracket = true
		case ']':
			if inBracket {
				if current != "" {
					keys = append(keys, current)
					current = ""
				}
				inBracket = false
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}

	// 添加最后的部分
	if current != "" {
		keys = append(keys, current)
	}

	return keys
}
