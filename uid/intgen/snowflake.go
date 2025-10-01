package intgen

import (
	"net"
	"sync/atomic"
	"time"
)

// Options 配置选项
type Options struct {
	MachineID *int64 // 机器ID，如果为nil则自动从IP地址获取
}

// SnowflakeGenerator Snowflake算法生成器
// 64位结构：1位符号位(0) + 41位时间戳 + 10位机器ID + 12位序列号
type SnowflakeGenerator struct {
	state     int64 // 原子状态：高52位时间戳 + 低12位序列号
	machineID int64 // 10位机器ID
	epoch     int64 // 起始纪元时间（毫秒）
}

const (
	sequenceBits  = 12
	machineIDBits = 10
	timestampBits = 41

	maxSequence  = (1 << sequenceBits) - 1  // 4095
	maxMachineID = (1 << machineIDBits) - 1 // 1023

	machineIDShift = sequenceBits
	timestampShift = sequenceBits + machineIDBits
)

// NewSnowflakeGenerator 创建Snowflake生成器
func NewSnowflakeGenerator(opts *Options) *SnowflakeGenerator {
	var machineID int64
	
	if opts != nil && opts.MachineID != nil {
		machineID = *opts.MachineID
	} else {
		machineID = getMachineIDFromIP()
	}
	
	// 确保机器ID在有效范围内
	machineID = machineID & maxMachineID
	
	// 使用固定的起始时间（2020-01-01 00:00:00 UTC）
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	
	now := time.Now().UnixMilli()
	return &SnowflakeGenerator{
		state:     (now - epoch) << sequenceBits, // 初始状态：当前时间戳，序列号为0
		machineID: machineID,
		epoch:     epoch,
	}
}

// getMachineIDFromIP 从IP地址获取机器ID
func getMachineIDFromIP() int64 {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 0
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				// 使用IPv4地址的最后两个字节作为机器ID
				return int64(ipv4[2])<<8 | int64(ipv4[3])
			}
		}
	}
	
	return 0
}

// Generate 生成Snowflake ID
func (g *SnowflakeGenerator) Generate() int64 {
	for {
		oldState := atomic.LoadInt64(&g.state)
		oldTimestamp := oldState >> sequenceBits
		oldSequence := oldState & maxSequence
		
		currentTimestamp := time.Now().UnixMilli() - g.epoch
		
		var newTimestamp, newSequence int64
		
		if currentTimestamp == oldTimestamp {
			// 同一毫秒内，序列号递增
			newSequence = (oldSequence + 1) & maxSequence
			if newSequence == 0 {
				// 序列号溢出，等待下一毫秒
				for currentTimestamp <= oldTimestamp {
					currentTimestamp = time.Now().UnixMilli() - g.epoch
				}
				newTimestamp = currentTimestamp
			} else {
				newTimestamp = oldTimestamp
			}
		} else {
			// 新的毫秒，序列号重置
			newTimestamp = currentTimestamp
			newSequence = 0
		}
		
		newState := (newTimestamp << sequenceBits) | newSequence
		
		// 原子更新状态
		if atomic.CompareAndSwapInt64(&g.state, oldState, newState) {
			// 组装最终的Snowflake ID
			return (newTimestamp << timestampShift) | (g.machineID << machineIDShift) | newSequence
		}
		// CAS失败，重试
	}
}