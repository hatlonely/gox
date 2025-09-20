package store

import (
	"context"
	"reflect"
	"time"

	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisStoreOptions struct {
	// host:port 地址。
	Endpoint string `cfg:"endpoint"`

	// 集群节点的 host:port 地址列表。
	Endpoints []string `cfg:"endpoints"`

	// 默认 TTL。
	DefaultTTL int `cfg:"defaultTTL" def:"0"`

	// 键的序列化选项。
	KeySerializer *ref.TypeOptions `cfg:"keySerializer"`

	// 值的序列化选项。
	ValSerializer *ref.TypeOptions `cfg:"valSerializer"`

	// 使用指定的用户名来验证当前连接，
	// 当连接到使用 Redis ACL 系统的 Redis 6.0 或更高版本实例时，
	// 该用户名必须与 ACL 列表中的一个连接定义匹配。
	Username string `cfg:"username"`

	// 可选密码。必须与 requirepass 服务器配置选项中指定的密码匹配
	// （如果连接到 Redis 5.0 或更低版本实例），
	// 或者与使用 Redis ACL 系统的 Redis 6.0 或更高版本实例连接时的用户密码匹配。
	Password string `cfg:"password"`

	// 连接到服务器后选择的数据库。
	DB int `cfg:"db" def:"0"`

	// 放弃前的最大重试次数。
	// 默认是 3 次重试；-1（不是 0）禁用重试。
	MaxRetries int `cfg:"maxRetries" def:"3"`

	// 每次重试之间的最小退避时间。
	// 默认是 8 毫秒；-1 禁用退避。
	MinRetryBackoff time.Duration `cfg:"minRetryBackoff" def:"8ms"`

	// 每次重试之间的最大退避时间。
	// 默认是 512 毫秒；-1 禁用退避。
	MaxRetryBackoff time.Duration `cfg:"maxRetryBackoff" def:"512ms"`

	// 建立新连接的拨号超时时间。
	// 默认是 5 秒。
	DialTimeout time.Duration `cfg:"dialTimeout" def:"5s"`

	// 套接字读取的超时时间。如果达到此时间，命令将失败，
	// 而不是阻塞。支持的值：
	//   - `0` - 默认超时时间（3 秒）。
	//   - `-1` - 无超时（无限期阻塞）。
	//   - `-2` - 完全禁用 SetReadDeadline 调用。
	ReadTimeout time.Duration `cfg:"readTimeout" def:"3s"`

	// 套接字写入的超时时间。如果达到此时间，命令将失败，
	// 而不是阻塞。支持的值：
	//   - `0` - 默认超时时间（3 秒）。
	//   - `-1` - 无超时（无限期阻塞）。
	//   - `-2` - 完全禁用 SetWriteDeadline 调用。
	WriteTimeout time.Duration `cfg:"writeTimeout" def:"3s"`

	// 连接池类型。
	// true 表示 FIFO 池，false 表示 LIFO 池。
	// 注意，FIFO 比 LIFO 有稍高的开销，
	// 但它有助于更快地关闭空闲连接，从而减少池的大小。
	PoolFIFO bool `cfg:"poolFIFO" def:"false"`

	// 基础的套接字连接数。
	// 默认是每个可用 CPU（由 runtime.GOMAXPROCS 报告）10 个连接。
	// 如果池中没有足够的连接，将会超出 PoolSize 分配新连接，
	// 你可以通过 MaxActiveConns 限制它。
	PoolSize int `cfg:"poolSize" def:"100"`

	// 如果所有连接都忙，客户端等待连接的时间，
	// 然后返回错误。
	// 默认是 ReadTimeout + 1 秒。
	PoolTimeout time.Duration `cfg:"poolTimeout" def:"4s"`

	// 最小空闲连接数，当建立新连接很慢时很有用。
	// 默认是 0。默认情况下空闲连接不会关闭。
	MinIdleConns int `cfg:"minIdleConns" def:"0"`

	// 最大空闲连接数。
	// 默认是 0。默认情况下空闲连接不会关闭。
	MaxIdleConns int `cfg:"maxIdleConns" def:"0"`

	// 池中在给定时间分配的最大连接数。
	// 当为零时，池中连接数没有限制。
	MaxActiveConns int `cfg:"maxActiveConns" def:"0"`

	// ConnMaxIdleTime 是连接可能空闲的最长时间。
	// 应小于服务器的超时时间。
	//
	// 过期的连接可能在重用之前被懒惰地关闭。
	// 如果 d <= 0，连接不会因空闲时间而关闭。
	//
	// 默认是 30 分钟。-1 禁用空闲超时检查。
	ConnMaxIdleTime time.Duration `cfg:"connMaxIdleTime" def:"30m"`

	// ConnMaxLifetime 是连接可能被重用的最长时间。
	//
	// 过期的连接可能在重用之前被懒惰地关闭。
	// 如果 <= 0，连接不会因连接的年龄而关闭。
	//
	// 默认是不关闭空闲连接。
	ConnMaxLifetime time.Duration `cfg:"connMaxLifetime" def:"0"`

	// 网络类型，可以是 tcp 或 unix。
	// 默认是 tcp。
	Network string `cfg:"network" def:"tcp"`

	// 放弃前的最大重试次数。命令在网络错误和 MOVED/ASK 重定向时会重试。
	// 默认是 3 次重试。
	MaxRedirects int `cfg:"maxRedirects" def:"3"`
}

type RedisStore[K, V any] struct {
	client redis.Cmdable

	keySerializer serializer.Serializer[K, []byte]
	valSerializer serializer.Serializer[V, []byte]
	defaultTTL    int
}

func NewRedisStoreWithOptions[K, V any](options *RedisStoreOptions) (*RedisStore[K, V], error) {
	// 注册当前泛型类型的序列化器
	ref.RegisterT[*serializer.JSONSerializer[K]](serializer.NewJSONSerializer[K])
	ref.RegisterT[*serializer.MsgPackSerializer[K]](serializer.NewMsgPackSerializer[K])
	ref.RegisterT[*serializer.BSONSerializer[K]](serializer.NewBSONSerializer[K])

	ref.RegisterT[*serializer.JSONSerializer[V]](serializer.NewJSONSerializer[V])
	ref.RegisterT[*serializer.MsgPackSerializer[V]](serializer.NewMsgPackSerializer[V])
	ref.RegisterT[*serializer.BSONSerializer[V]](serializer.NewBSONSerializer[V])

	// 获取K和V的类型名，用于构造默认TypeOptions
	var k K
	var v V

	// 设置默认的序列化器配置
	keySerializerOptions := options.KeySerializer
	if keySerializerOptions == nil {
		keySerializerOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/serializer",
			Type:      "MsgPackSerializer[" + reflect.TypeOf(k).String() + "]",
		}
	}

	valSerializerOptions := options.ValSerializer
	if valSerializerOptions == nil {
		valSerializerOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/serializer",
			Type:      "MsgPackSerializer[" + reflect.TypeOf(v).String() + "]",
		}
	}

	// 构造 key 序列化器
	keySerializerInterface, err := ref.NewWithOptions(keySerializerOptions)
	if err != nil {
		return nil, err
	}
	keySerializer, ok := keySerializerInterface.(serializer.Serializer[K, []byte])
	if !ok {
		return nil, errors.New("invalid key serializer type")
	}

	// 构造 value 序列化器
	valSerializerInterface, err := ref.NewWithOptions(valSerializerOptions)
	if err != nil {
		return nil, err
	}
	valSerializer, ok := valSerializerInterface.(serializer.Serializer[V, []byte])
	if !ok {
		return nil, errors.New("invalid value serializer type")
	}

	var client redis.Cmdable

	if options.Endpoint != "" {
		client = redis.NewClient(&redis.Options{
			Addr:            options.Endpoint,
			Username:        options.Username,
			Password:        options.Password,
			DB:              options.DB,
			MaxRetries:      options.MaxRetries,
			MinRetryBackoff: options.MinRetryBackoff,
			MaxRetryBackoff: options.MaxRetryBackoff,
			DialTimeout:     options.DialTimeout,
			ReadTimeout:     options.ReadTimeout,
			WriteTimeout:    options.WriteTimeout,
			PoolFIFO:        options.PoolFIFO,
			PoolSize:        options.PoolSize,
			PoolTimeout:     options.PoolTimeout,
			MinIdleConns:    options.MinIdleConns,
			MaxIdleConns:    options.MaxIdleConns,
			MaxActiveConns:  options.MaxActiveConns,
			ConnMaxIdleTime: options.ConnMaxIdleTime,
			ConnMaxLifetime: options.ConnMaxLifetime,
			Network:         options.Network,
		})
	} else if len(options.Endpoints) > 0 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           options.Endpoints,
			Username:        options.Username,
			Password:        options.Password,
			MaxRetries:      options.MaxRetries,
			DialTimeout:     options.DialTimeout,
			ReadTimeout:     options.ReadTimeout,
			WriteTimeout:    options.WriteTimeout,
			PoolFIFO:        options.PoolFIFO,
			PoolSize:        options.PoolSize,
			PoolTimeout:     options.PoolTimeout,
			MinIdleConns:    options.MinIdleConns,
			MaxIdleConns:    options.MaxIdleConns,
			MaxActiveConns:  options.MaxActiveConns,
			ConnMaxIdleTime: options.ConnMaxIdleTime,
			ConnMaxLifetime: options.ConnMaxLifetime,
			MaxRedirects:    options.MaxRedirects,
		})
	} else {
		return nil, errors.Errorf("Endpoint or Endpoints must be set")
	}

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, errors.WithMessage(err, "redis.client.Ping failed")
	}

	return &RedisStore[K, V]{
		client:        client,
		keySerializer: keySerializer,
		valSerializer: valSerializer,
		defaultTTL:    options.DefaultTTL,
	}, nil
}
