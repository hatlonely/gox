package store

import (
	"context"
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
	// 构造 key 序列化器
	keySerializer, err := serializer.NewByteSerializerWithOptions[K](options.KeySerializer)
	if err != nil {
		return nil, err
	}

	// 构造 value 序列化器
	valSerializer, err := serializer.NewByteSerializerWithOptions[V](options.ValSerializer)
	if err != nil {
		return nil, err
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

func (s *RedisStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return err
	}

	valueBytes, err := s.valSerializer.Serialize(value)
	if err != nil {
		return err
	}

	keyStr := string(keyBytes)

	if options.IfNotExist {
		exists, err := s.client.Exists(ctx, keyStr).Result()
		if err != nil {
			return err
		}
		if exists > 0 {
			return ErrConditionFailed
		}
	}

	expiration := time.Duration(options.Expiration)
	if expiration == 0 && s.defaultTTL > 0 {
		expiration = time.Duration(s.defaultTTL) * time.Second
	}

	return s.client.Set(ctx, keyStr, valueBytes, expiration).Err()
}

func (s *RedisStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return zero, err
	}

	keyStr := string(keyBytes)
	valueBytes, err := s.client.Get(ctx, keyStr).Bytes()
	if err != nil {
		if err == redis.Nil {
			return zero, ErrKeyNotFound
		}
		return zero, err
	}

	return s.valSerializer.Deserialize(valueBytes)
}

func (s *RedisStore[K, V]) Del(ctx context.Context, key K) error {
	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return err
	}

	keyStr := string(keyBytes)
	return s.client.Del(ctx, keyStr).Err()
}

func (s *RedisStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, errors.New("keys and vals length mismatch")
	}

	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	errs := make([]error, len(keys))
	
	// 预处理：序列化所有键值对
	type keyValuePair struct {
		keyStr   string
		keyBytes []byte
		valueBytes []byte
		index    int
	}
	
	var pairs []keyValuePair
	for i := range keys {
		keyBytes, err := s.keySerializer.Serialize(keys[i])
		if err != nil {
			errs[i] = err
			continue
		}

		valueBytes, err := s.valSerializer.Serialize(vals[i])
		if err != nil {
			errs[i] = err
			continue
		}

		pairs = append(pairs, keyValuePair{
			keyStr:     string(keyBytes),
			keyBytes:   keyBytes,
			valueBytes: valueBytes,
			index:      i,
		})
	}

	// 如果需要检查 IfNotExist，先批量检查键是否存在
	if options.IfNotExist {
		keyStrs := make([]string, len(pairs))
		for i, pair := range pairs {
			keyStrs[i] = pair.keyStr
		}
		
		// 使用 MGET 批量检查键是否存在
		results, err := s.client.MGet(ctx, keyStrs...).Result()
		if err != nil {
			// 如果批量检查失败，回退到单个检查
			for _, pair := range pairs {
				exists, checkErr := s.client.Exists(ctx, pair.keyStr).Result()
				if checkErr != nil {
					errs[pair.index] = checkErr
					continue
				}
				if exists > 0 {
					errs[pair.index] = ErrConditionFailed
				}
			}
		} else {
			// 处理 MGET 结果
			for i, result := range results {
				if i < len(pairs) && result != nil {
					errs[pairs[i].index] = ErrConditionFailed
				}
			}
		}
		
		// 过滤掉已经存在的键
		var validPairs []keyValuePair
		for _, pair := range pairs {
			if errs[pair.index] == nil {
				validPairs = append(validPairs, pair)
			}
		}
		pairs = validPairs
	}

	// 使用 Pipeline 批量执行 SET 命令
	if len(pairs) > 0 {
		pipe := s.client.Pipeline()
		
		expiration := time.Duration(options.Expiration)
		if expiration == 0 && s.defaultTTL > 0 {
			expiration = time.Duration(s.defaultTTL) * time.Second
		}

		for _, pair := range pairs {
			pipe.Set(ctx, pair.keyStr, pair.valueBytes, expiration)
		}

		// 执行 Pipeline
		cmds, err := pipe.Exec(ctx)
		if err != nil {
			// 如果 Pipeline 执行失败，记录错误
			for _, pair := range pairs {
				if errs[pair.index] == nil {
					errs[pair.index] = err
				}
			}
		} else {
			// 检查每个命令的执行结果
			for i, cmd := range cmds {
				if i < len(pairs) {
					if cmdErr := cmd.Err(); cmdErr != nil && cmdErr != redis.Nil {
						errs[pairs[i].index] = cmdErr
					}
				}
			}
		}
	}

	return errs, nil
}

func (s *RedisStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	vals := make([]V, len(keys))
	errs := make([]error, len(keys))

	if len(keys) == 0 {
		return vals, errs, nil
	}

	// 序列化所有键
	keyStrs := make([]string, 0, len(keys))
	keyIndexMap := make(map[int]int) // 原始索引到有效索引的映射
	validCount := 0

	for i, key := range keys {
		keyBytes, err := s.keySerializer.Serialize(key)
		if err != nil {
			errs[i] = err
			var zero V
			vals[i] = zero
			continue
		}
		
		keyStrs = append(keyStrs, string(keyBytes))
		keyIndexMap[i] = validCount
		validCount++
	}

	// 如果没有有效的键，直接返回
	if len(keyStrs) == 0 {
		return vals, errs, nil
	}

	// 使用 MGET 批量获取
	results, err := s.client.MGet(ctx, keyStrs...).Result()
	if err != nil {
		// 如果 MGET 失败，回退到单个 GET
		validIndex := 0
		for i, key := range keys {
			if errs[i] != nil {
				continue // 跳过序列化失败的键
			}
			
			val, getErr := s.Get(ctx, key)
			vals[i] = val
			errs[i] = getErr
			validIndex++
		}
		return vals, errs, nil
	}

	// 处理 MGET 结果
	validIndex := 0
	for i := range keys {
		if errs[i] != nil {
			continue // 跳过序列化失败的键
		}

		if validIndex < len(results) {
			result := results[validIndex]
			if result == nil {
				// 键不存在
				var zero V
				vals[i] = zero
				errs[i] = ErrKeyNotFound
			} else {
				// 反序列化值
				valueBytes := []byte(result.(string))
				value, deserErr := s.valSerializer.Deserialize(valueBytes)
				if deserErr != nil {
					var zero V
					vals[i] = zero
					errs[i] = deserErr
				} else {
					vals[i] = value
					errs[i] = nil
				}
			}
		}
		validIndex++
	}

	return vals, errs, nil
}

func (s *RedisStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errs := make([]error, len(keys))
	
	if len(keys) == 0 {
		return errs, nil
	}

	// 序列化所有键
	type keyInfo struct {
		keyStr string
		index  int
	}
	
	var validKeys []keyInfo
	for i, key := range keys {
		keyBytes, err := s.keySerializer.Serialize(key)
		if err != nil {
			errs[i] = err
			continue
		}
		
		validKeys = append(validKeys, keyInfo{
			keyStr: string(keyBytes),
			index:  i,
		})
	}

	// 如果没有有效的键，直接返回
	if len(validKeys) == 0 {
		return errs, nil
	}

	// 使用 Pipeline 批量删除
	pipe := s.client.Pipeline()
	
	for _, keyInfo := range validKeys {
		pipe.Del(ctx, keyInfo.keyStr)
	}

	// 执行 Pipeline
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		// 如果 Pipeline 执行失败，记录错误
		for _, keyInfo := range validKeys {
			errs[keyInfo.index] = err
		}
	} else {
		// 检查每个命令的执行结果
		for i, cmd := range cmds {
			if i < len(validKeys) {
				if cmdErr := cmd.Err(); cmdErr != nil && cmdErr != redis.Nil {
					errs[validKeys[i].index] = cmdErr
				}
			}
		}
	}

	return errs, nil
}

func (s *RedisStore[K, V]) Close() error {
	if client, ok := s.client.(*redis.Client); ok {
		return client.Close()
	}
	if clusterClient, ok := s.client.(*redis.ClusterClient); ok {
		return clusterClient.Close()
	}
	return nil
}
