package intgen

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions Redis生成器配置选项
type RedisOptions struct {
	Addr     string        // Redis地址，默认 "localhost:6379"
	Password string        // 密码
	DB       int           // 数据库编号，默认 0
	KeyName  string        // 存储序列号的键名，默认 "uid:sequence"
	Timeout  time.Duration // 操作超时时间，默认 3秒
}

// RedisGenerator 基于Redis的分布式ID生成器
// 使用时间戳+序列号模式，序列号存储在Redis中保证分布式唯一性
type RedisGenerator struct {
	client  *redis.Client
	keyName string
	timeout time.Duration
}

// NewRedisGeneratorWithOptions 创建Redis生成器
func NewRedisGeneratorWithOptions(options *RedisOptions) *RedisGenerator {
	if options == nil {
		options = &RedisOptions{}
	}

	// 设置默认值
	if options.Addr == "" {
		options.Addr = "localhost:6379"
	}
	if options.KeyName == "" {
		options.KeyName = "uid:sequence"
	}
	if options.Timeout == 0 {
		options.Timeout = 3 * time.Second
	}

	client := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password,
		DB:       options.DB,
	})

	return &RedisGenerator{
		client:  client,
		keyName: options.KeyName,
		timeout: options.Timeout,
	}
}

// Generate 生成ID：高52位时间戳(毫秒) + 低12位序列号
// 使用Redis INCR命令保证分布式环境下序列号的唯一性
func (g *RedisGenerator) Generate() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 获取当前时间戳(毫秒)
	timestamp := time.Now().UnixMilli()

	// 构造Redis键：keyName:timestamp
	key := g.keyName + ":" + formatTimestamp(timestamp)

	// 使用Redis INCR原子操作获取序列号
	sequence, err := g.client.Incr(ctx, key).Result()
	if err != nil {
		// Redis操作失败，使用本地时间戳作为降级方案
		return timestamp << 12
	}

	// 设置键的过期时间为2秒，避免Redis中积累过多键
	g.client.Expire(ctx, key, 2*time.Second)

	// 序列号限制在12位范围内 (0-4095)
	seq := (sequence - 1) & 0xFFF

	// 如果序列号溢出，等待下一毫秒
	if seq == 0 && sequence > 1 {
		time.Sleep(time.Millisecond)
		return g.Generate()
	}

	// 组装最终ID：52位时间戳 + 12位序列号
	return (timestamp << 12) | seq
}

// formatTimestamp 格式化时间戳为字符串
func formatTimestamp(timestamp int64) string {
	return time.Unix(0, timestamp*int64(time.Millisecond)).Format("20060102150405.000")
}