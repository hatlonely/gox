package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiStorage(t *testing.T) {
	t.Run("创建空的MultiStorage", func(t *testing.T) {
		ms := NewMultiStorage(nil)
		require.NotNil(t, ms)

		// 空存储应该能正常工作
		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)
	})

	t.Run("创建带有存储源的MultiStorage", func(t *testing.T) {
		source1 := NewMapStorage(map[string]interface{}{
			"key1": "value1",
		})
		source2 := NewMapStorage(map[string]interface{}{
			"key2": "value2",
		})

		ms := NewMultiStorage([]Storage{source1, source2})
		require.NotNil(t, ms)

		// 验证不会被外部修改影响
		originalSources := []Storage{source1, source2}
		originalSources[0] = nil // 修改原始切片
		
		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
	})
}

func TestMultiStorage_ConvertTo(t *testing.T) {
	t.Run("单个存储源", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"name": "test",
			"port": 8080,
		})

		ms := NewMultiStorage([]Storage{source})

		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, "test", result["name"])
		assert.Equal(t, 8080, result["port"])
	})

	t.Run("多个存储源按优先级合并", func(t *testing.T) {
		// 基础配置
		base := NewMapStorage(map[string]interface{}{
			"name": "base",
			"port": 8080,
			"debug": false,
		})

		// 覆盖配置
		override := NewMapStorage(map[string]interface{}{
			"name": "override",
			"port": 9090,
		})

		ms := NewMultiStorage([]Storage{base, override})

		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)

		// 验证覆盖规则：后面的配置覆盖前面的配置
		assert.Equal(t, "override", result["name"]) // 被覆盖
		assert.Equal(t, 9090, result["port"])       // 被覆盖
		assert.Equal(t, false, result["debug"])     // 保留原值
	})

	t.Run("包含nil存储源", func(t *testing.T) {
		source1 := NewMapStorage(map[string]interface{}{
			"key1": "value1",
		})

		ms := NewMultiStorage([]Storage{source1, nil})

		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
	})

	t.Run("结构体转换", func(t *testing.T) {
		type Config struct {
			Name    string `cfg:"name"`
			Port    int    `cfg:"port"`
			Debug   bool   `cfg:"debug"`
			Feature string `cfg:"feature"`
		}

		base := NewMapStorage(map[string]interface{}{
			"name":    "app",
			"port":    8080,
			"debug":   false,
			"feature": "base-feature",
		})

		env := NewMapStorage(map[string]interface{}{
			"port":    9090,
			"debug":   true,
			"feature": "env-feature",
		})

		ms := NewMultiStorage([]Storage{base, env})

		var config Config
		err := ms.ConvertTo(&config)
		assert.NoError(t, err)
		
		// 验证合并结果
		assert.Equal(t, "app", config.Name)           // base 的值
		assert.Equal(t, 9090, config.Port)           // env 覆盖
		assert.Equal(t, true, config.Debug)          // env 覆盖
		assert.Equal(t, "env-feature", config.Feature) // env 覆盖
	})

	t.Run("nil参数", func(t *testing.T) {
		ms := NewMultiStorage([]Storage{})
		err := ms.ConvertTo(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object cannot be nil")
	})
}

func TestMultiStorage_Sub(t *testing.T) {
	t.Run("获取子配置", func(t *testing.T) {
		base := NewMapStorage(map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
		})

		override := NewMapStorage(map[string]interface{}{
			"database": map[string]interface{}{
				"port": 3306,
				"name": "test",
			},
		})

		ms := NewMultiStorage([]Storage{base, override})
		dbConfig := ms.Sub("database")

		var result map[string]interface{}
		err := dbConfig.ConvertTo(&result)
		assert.NoError(t, err)

		// 验证子配置的合并
		assert.Equal(t, "localhost", result["host"]) // base 的值
		assert.Equal(t, 3306, result["port"])        // override 覆盖
		assert.Equal(t, "test", result["name"])      // override 的值
	})

	t.Run("空键返回自身", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"key": "value",
		})
		ms := NewMultiStorage([]Storage{source})

		sub := ms.Sub("")
		
		// 应该能获取到相同的数据
		var original, subResult map[string]interface{}
		ms.ConvertTo(&original)
		sub.ConvertTo(&subResult)
		
		assert.Equal(t, original, subResult)
	})

	t.Run("不存在的键", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"existing": "value",
		})
		ms := NewMultiStorage([]Storage{source})

		nonExistentSub := ms.Sub("non-existent")
		
		var result map[string]interface{}
		err := nonExistentSub.ConvertTo(&result)
		assert.NoError(t, err)
		// result 应该为空或保持原样
	})
}

func TestMultiStorage_UpdateStorage(t *testing.T) {
	t.Run("更新有效索引", func(t *testing.T) {
		original := NewMapStorage(map[string]interface{}{
			"key": "original",
		})
		ms := NewMultiStorage([]Storage{original})

		updated := NewMapStorage(map[string]interface{}{
			"key": "updated",
		})

		// 更新存储源
		changed := ms.UpdateStorage(0, updated)
		assert.True(t, changed)

		// 验证更新结果
		var result map[string]interface{}
		err := ms.ConvertTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, "updated", result["key"])
	})

	t.Run("更新无效索引", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"key": "value",
		})
		ms := NewMultiStorage([]Storage{source})

		newSource := NewMapStorage(map[string]interface{}{
			"key": "new",
		})

		// 无效索引应该返回 false
		assert.False(t, ms.UpdateStorage(-1, newSource))
		assert.False(t, ms.UpdateStorage(1, newSource))
	})

	t.Run("相同内容不算变更", func(t *testing.T) {
		original := NewMapStorage(map[string]interface{}{
			"key": "value",
		})
		ms := NewMultiStorage([]Storage{original})

		// 创建相同内容的存储
		same := NewMapStorage(map[string]interface{}{
			"key": "value",
		})

		changed := ms.UpdateStorage(0, same)
		assert.False(t, changed) // 相同内容，不算变更
	})

	t.Run("nil值处理", func(t *testing.T) {
		ms := NewMultiStorage([]Storage{nil})

		// nil -> storage 算变更
		newStorage := NewMapStorage(map[string]interface{}{"key": "value"})
		assert.True(t, ms.UpdateStorage(0, newStorage))

		// storage -> nil 也算变更
		assert.True(t, ms.UpdateStorage(0, nil))

		// nil -> nil 不算变更
		assert.False(t, ms.UpdateStorage(0, nil))
	})
}

func TestMultiStorage_Equals(t *testing.T) {
	t.Run("相同的MultiStorage", func(t *testing.T) {
		source1 := NewMapStorage(map[string]interface{}{
			"key1": "value1",
		})
		source2 := NewMapStorage(map[string]interface{}{
			"key2": "value2",
		})

		ms1 := NewMultiStorage([]Storage{source1, source2})
		ms2 := NewMultiStorage([]Storage{source1, source2})

		assert.True(t, ms1.Equals(ms2))
	})

	t.Run("不同的MultiStorage", func(t *testing.T) {
		source1 := NewMapStorage(map[string]interface{}{
			"key1": "value1",
		})
		source2 := NewMapStorage(map[string]interface{}{
			"key2": "value2",
		})
		source3 := NewMapStorage(map[string]interface{}{
			"key3": "value3",
		})

		ms1 := NewMultiStorage([]Storage{source1, source2})
		ms2 := NewMultiStorage([]Storage{source1, source3})

		assert.False(t, ms1.Equals(ms2))
	})

	t.Run("不同数量的存储源", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"key": "value",
		})

		ms1 := NewMultiStorage([]Storage{source})
		ms2 := NewMultiStorage([]Storage{source, source})

		assert.False(t, ms1.Equals(ms2))
	})

	t.Run("nil值比较", func(t *testing.T) {
		ms := NewMultiStorage([]Storage{})
		
		assert.False(t, ms.Equals(nil))
	})

	t.Run("不同类型的Storage", func(t *testing.T) {
		source := NewMapStorage(map[string]interface{}{
			"key": "value",
		})
		ms := NewMultiStorage([]Storage{source})

		// 与普通 MapStorage 比较
		assert.False(t, ms.Equals(source))
	})
}