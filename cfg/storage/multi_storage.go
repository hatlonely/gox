package storage

import (
	"fmt"
	"sync"
)

// MultiStorage 多配置存储，支持多个存储源按优先级合并
// 实现 Storage 接口，提供统一的配置访问入口
type MultiStorage interface {
	Storage

	// UpdateStorage 更新指定索引的存储源，返回是否有变更
	UpdateStorage(index int, storage Storage) bool
}

// multiStorage 多配置存储的具体实现
type multiStorage struct {
	sources []Storage    // 配置源存储数组，索引越大优先级越高
	mu      sync.RWMutex // 读写锁，保护并发访问
}

// NewMultiStorage 创建多配置存储
// sources: 配置源数组，按优先级排序（索引越大优先级越高）
func NewMultiStorage(sources []Storage) MultiStorage {
	if sources == nil {
		sources = make([]Storage, 0)
	}

	// 复制切片，避免外部修改
	sourcesCopy := make([]Storage, len(sources))
	copy(sourcesCopy, sources)

	return &multiStorage{
		sources: sourcesCopy,
	}
}

// UpdateStorage 更新指定索引的存储源，返回是否有变更
func (ms *multiStorage) UpdateStorage(index int, storage Storage) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// 检查索引有效性
	if index < 0 || index >= len(ms.sources) {
		return false
	}

	// 检测是否有变更
	old := ms.sources[index]
	if old != nil && storage != nil && old.Equals(storage) {
		return false // 没有变更
	}
	if old == nil && storage == nil {
		return false // 都为 nil，没有变更
	}

	// 更新存储源
	ms.sources[index] = storage
	return true // 有变更
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
// 按照优先级顺序依次调用每个存储源的 ConvertTo，实现增量合并
func (ms *multiStorage) ConvertTo(object any) error {
	if object == nil {
		return fmt.Errorf("object cannot be nil")
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// 依次调用每个存储源的 ConvertTo，实现增量合并
	// - 对于结构体：字段级覆盖，后面的配置覆盖前面的配置，不存在的字段保持原值
	// - 对于 map：增量合并，新键被添加，已存在的键被覆盖，其他键被保留
	// - 对于其他类型：按照各 Storage 实现的语义处理
	for i, storage := range ms.sources {
		if storage != nil {
			if err := storage.ConvertTo(object); err != nil {
				return fmt.Errorf("failed to convert from source %d: %w", i, err)
			}
		}
	}

	return nil
}


// Sub 获取子配置存储对象
// 对每个存储源调用 Sub，然后创建新的 MultiStorage
func (ms *multiStorage) Sub(key string) Storage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// 为每个存储源创建对应的子存储
	subSources := make([]Storage, len(ms.sources))
	for i, storage := range ms.sources {
		if storage != nil {
			subSources[i] = storage.Sub(key)
		}
		// 如果 storage 为 nil，subSources[i] 保持为 nil
	}

	return NewMultiStorage(subSources)
}

// Equals 比较两个存储是否包含相同的数据内容
func (ms *multiStorage) Equals(other Storage) bool {
	if other == nil {
		return ms == nil
	}

	// 类型检查
	otherMulti, ok := other.(*multiStorage)
	if !ok {
		return false
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	otherMulti.mu.RLock()
	defer otherMulti.mu.RUnlock()

	// 检查存储源数量
	if len(ms.sources) != len(otherMulti.sources) {
		return false
	}

	// 逐个比较存储源
	for i, source := range ms.sources {
		otherSource := otherMulti.sources[i]

		// nil 值比较
		if source == nil && otherSource == nil {
			continue
		}
		if source == nil || otherSource == nil {
			return false
		}

		// 调用存储源的 Equals 方法比较
		if !source.Equals(otherSource) {
			return false
		}
	}

	return true
}