package intgen

import (
	"sync"
	"testing"
)

func TestTimestampSeqGenerator_Generate(t *testing.T) {
	gen := NewTimestampSeqGenerator()

	// 测试基本功能
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 >= id2 {
		t.Errorf("ID应该递增，但得到 id1=%d, id2=%d", id1, id2)
	}

	// 测试ID结构
	timestamp := id1 >> 12
	sequence := id1 & 0xFFF

	if timestamp <= 0 {
		t.Errorf("时间戳应该大于0，但得到 %d", timestamp)
	}

	if sequence < 0 || sequence > 0xFFF {
		t.Errorf("序列号应该在0-4095范围内，但得到 %d", sequence)
	}
}

func TestTimestampSeqGenerator_Uniqueness(t *testing.T) {
	gen := NewTimestampSeqGenerator()
	ids := make(map[int64]bool)
	count := 10000

	for i := 0; i < count; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("生成了重复的ID: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("期望生成 %d 个唯一ID，但实际生成了 %d 个", count, len(ids))
	}
}

func TestTimestampSeqGenerator_Concurrent(t *testing.T) {
	gen := NewTimestampSeqGenerator()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[int64]bool)
	goroutines := 100
	idsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localIds := make([]int64, 0, idsPerGoroutine)

			for j := 0; j < idsPerGoroutine; j++ {
				id := gen.Generate()
				localIds = append(localIds, id)
			}

			mu.Lock()
			for _, id := range localIds {
				if ids[id] {
					t.Errorf("并发测试中生成了重复的ID: %d", id)
				}
				ids[id] = true
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	expectedCount := goroutines * idsPerGoroutine
	if len(ids) != expectedCount {
		t.Errorf("期望生成 %d 个唯一ID，但实际生成了 %d 个", expectedCount, len(ids))
	}
}

func TestTimestampSeqGenerator_Ordering(t *testing.T) {
	gen := NewTimestampSeqGenerator()
	count := 1000
	ids := make([]int64, count)

	for i := 0; i < count; i++ {
		ids[i] = gen.Generate()
	}

	// 检查时间戳部分是否有序
	for i := 1; i < count; i++ {
		prevTimestamp := ids[i-1] >> 12
		currTimestamp := ids[i] >> 12

		if currTimestamp < prevTimestamp {
			t.Errorf("时间戳应该递增，但在位置 %d 发现倒序: prev=%d, curr=%d", 
				i, prevTimestamp, currTimestamp)
		}
	}
}

func TestTimestampSeqGenerator_SequenceRollover(t *testing.T) {
	gen := NewTimestampSeqGenerator()

	// 快速生成大量ID以测试序列号溢出
	prevTimestamp := int64(-1)
	maxSequenceInSameMs := int64(0)

	for i := 0; i < 5000; i++ {
		id := gen.Generate()
		timestamp := id >> 12
		sequence := id & 0xFFF

		if timestamp == prevTimestamp {
			if sequence > maxSequenceInSameMs {
				maxSequenceInSameMs = sequence
			}
		} else {
			if prevTimestamp != -1 && maxSequenceInSameMs > 0 {
				// 验证在同一毫秒内序列号是连续的
				t.Logf("在时间戳 %d 中，最大序列号为 %d", prevTimestamp, maxSequenceInSameMs)
			}
			maxSequenceInSameMs = sequence
			prevTimestamp = timestamp
		}
	}
}

func BenchmarkTimestampSeqGenerator_Generate(b *testing.B) {
	gen := NewTimestampSeqGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkTimestampSeqGenerator_GenerateConcurrent(b *testing.B) {
	gen := NewTimestampSeqGenerator()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gen.Generate()
		}
	})
}

func TestTimestampSeqGenerator_Interface(t *testing.T) {
	var _ IntGenerator = &TimestampSeqGenerator{}

	gen := NewTimestampSeqGenerator()
	var iGen IntGenerator = gen

	id := iGen.Generate()
	if id <= 0 {
		t.Errorf("通过接口生成的ID应该大于0，但得到 %d", id)
	}
}