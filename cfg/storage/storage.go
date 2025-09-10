package storage

// Storage 配置数据存储接口
// 提供层级化配置访问和结构体绑定功能
type Storage interface {
	// Sub 获取子配置存储对象
	// key 可以包含点号（.）表示多级嵌套，[]表示数组索引
	// 例如 "database.connections[0].host"
	Sub(key string) Storage

	// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
	ConvertTo(object interface{}) error
}