package intgen

import (
	"sync/atomic"
	"time"
)

// TimestampSeqGenerator 时间戳+序列号生成器（使用原子操作）
type TimestampSeqGenerator struct {
	state int64 // 高52位：时间戳，低12位：序列号
}

// NewTimestampSeqGenerator 创建时间戳+序列号生成器
func NewTimestampSeqGenerator() *TimestampSeqGenerator {
	now := time.Now().UnixMilli()
	return &TimestampSeqGenerator{
		state: now << 12, // 初始状态：当前时间戳，序列号为0
	}
}

// Generate 生成ID：高52位时间戳(毫秒) + 低12位序列号
func (g *TimestampSeqGenerator) Generate() int64 {
	for {
		oldState := atomic.LoadInt64(&g.state)
		oldTimestamp := oldState >> 12
		oldSequence := oldState & 0xFFF

		currentTimestamp := time.Now().UnixMilli()

		var newTimestamp, newSequence int64

		if currentTimestamp == oldTimestamp {
			// 同一毫秒内，序列号递增
			newSequence = (oldSequence + 1) & 0xFFF
			if newSequence == 0 {
				// 序列号溢出，等待下一毫秒
				for currentTimestamp <= oldTimestamp {
					currentTimestamp = time.Now().UnixMilli()
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

		newState := (newTimestamp << 12) | newSequence

		// 原子更新状态
		if atomic.CompareAndSwapInt64(&g.state, oldState, newState) {
			return newState
		}
		// CAS失败，重试
	}
}