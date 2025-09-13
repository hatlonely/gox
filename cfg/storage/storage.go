package storage

// Storage 配置数据存储接口
// 提供层级化配置访问和结构体绑定功能
type Storage interface {
	// Sub 获取子配置存储对象
	// key 可以包含点号（.）表示多级嵌套，[]表示数组索引
	// 例如 "database.connections[0].host"
	//
	// 重要行为：
	// - 如果请求的 key 不存在，返回 nil Storage（类型化的 nil，如 *MapStorage(nil)）
	// - 返回的 nil Storage 可以安全调用 ConvertTo 方法，不会修改目标对象
	// - 这种设计支持链式调用和优雅的缺失配置处理
	Sub(key string) Storage

	// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
	// 支持智能指针字段处理和 nil Storage 处理
	//
	// 智能指针字段处理规则：
	// - 如果配置中没有该字段：保持指针字段的原始状态（nil 保持 nil，非 nil 保持不变）
	// - 如果配置中存在该字段：即使指针字段为 nil，也会创建新实例并赋值
	//
	// Nil Storage 处理：
	// - 如果 Storage 本身为 nil，ConvertTo 不做任何修改，直接返回 nil
	// - 这确保了空指针参数能保持空值状态
	//
	// 支持的标签优先级（从高到低）：
	// cfg > json > yaml > toml > ini > 字段名
	ConvertTo(object interface{}) error

	// Equals 比较两个 Storage 是否包含相同的数据内容
	// 各个实现可以根据自身特点优化比较逻辑
	//
	// Nil Storage 比较规则：
	// - nil Storage == nil Storage → true
	// - nil Storage == nil interface → false  
	// - nil Storage == 正常 Storage → false
	// - 正常 Storage == nil Storage → false
	Equals(other Storage) bool
}