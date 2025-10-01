package intgen

import (
	"sync"
	"testing"
)

func TestSnowflakeGenerator_Generate(t *testing.T) {
	// 测试使用默认配置（自动获取机器ID）
	gen := NewSnowflakeGenerator(nil)
	
	id1 := gen.Generate()
	id2 := gen.Generate()
	
	if id1 >= id2 {
		t.Errorf("ID应该递增，但得到 id1=%d, id2=%d", id1, id2)
	}
	
	// 验证ID结构
	timestamp := id1 >> timestampShift
	machineID := (id1 >> machineIDShift) & maxMachineID
	sequence := id1 & maxSequence
	
	if timestamp <= 0 {
		t.Errorf("时间戳应该大于0，但得到 %d", timestamp)
	}
	
	if machineID < 0 || machineID > maxMachineID {
		t.Errorf("机器ID应该在0-%d范围内，但得到 %d", maxMachineID, machineID)
	}
	
	if sequence < 0 || sequence > maxSequence {
		t.Errorf("序列号应该在0-%d范围内，但得到 %d", maxSequence, sequence)
	}
}

func TestSnowflakeGenerator_WithCustomMachineID(t *testing.T) {
	customMachineID := int64(123)
	opts := &Options{MachineID: &customMachineID}
	gen := NewSnowflakeGenerator(opts)
	
	id := gen.Generate()
	machineID := (id >> machineIDShift) & maxMachineID
	
	if machineID != customMachineID {
		t.Errorf("期望机器ID为 %d，但得到 %d", customMachineID, machineID)
	}
}

func TestSnowflakeGenerator_MachineIDRange(t *testing.T) {
	// 测试超出范围的机器ID会被截断
	largeMachineID := int64(2048) // 超出10位范围
	opts := &Options{MachineID: &largeMachineID}
	gen := NewSnowflakeGenerator(opts)
	
	id := gen.Generate()
	machineID := (id >> machineIDShift) & maxMachineID
	
	expectedMachineID := largeMachineID & maxMachineID
	if machineID != expectedMachineID {
		t.Errorf("期望机器ID为 %d（截断后），但得到 %d", expectedMachineID, machineID)
	}
}

func TestSnowflakeGenerator_Uniqueness(t *testing.T) {
	gen := NewSnowflakeGenerator(nil)
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

func TestSnowflakeGenerator_Concurrent(t *testing.T) {
	gen := NewSnowflakeGenerator(nil)
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

func TestSnowflakeGenerator_Ordering(t *testing.T) {
	gen := NewSnowflakeGenerator(nil)
	count := 1000
	ids := make([]int64, count)
	
	for i := 0; i < count; i++ {
		ids[i] = gen.Generate()
	}
	
	// 检查时间戳部分是否有序
	for i := 1; i < count; i++ {
		prevTimestamp := ids[i-1] >> timestampShift
		currTimestamp := ids[i] >> timestampShift
		
		if currTimestamp < prevTimestamp {
			t.Errorf("时间戳应该递增，但在位置 %d 发现倒序: prev=%d, curr=%d", 
				i, prevTimestamp, currTimestamp)
		}
	}
}

func TestSnowflakeGenerator_SequenceRollover(t *testing.T) {
	gen := NewSnowflakeGenerator(nil)
	
	// 快速生成大量ID以测试序列号溢出
	prevTimestamp := int64(-1)
	maxSequenceInSameMs := int64(0)
	
	for i := 0; i < 5000; i++ {
		id := gen.Generate()
		timestamp := id >> timestampShift
		sequence := id & maxSequence
		
		if timestamp == prevTimestamp {
			if sequence > maxSequenceInSameMs {
				maxSequenceInSameMs = sequence
			}
		} else {
			if prevTimestamp != -1 && maxSequenceInSameMs > 0 {
				t.Logf("在时间戳 %d 中，最大序列号为 %d", prevTimestamp, maxSequenceInSameMs)
			}
			maxSequenceInSameMs = sequence
			prevTimestamp = timestamp
		}
	}
}

func TestSnowflakeGenerator_Interface(t *testing.T) {
	var _ IntGenerator = &SnowflakeGenerator{}
	
	gen := NewSnowflakeGenerator(nil)
	var iGen IntGenerator = gen
	
	id := iGen.Generate()
	if id <= 0 {
		t.Errorf("通过接口生成的ID应该大于0，但得到 %d", id)
	}
}

func TestSnowflakeGenerator_DifferentMachines(t *testing.T) {
	// 测试不同机器ID生成的ID不会冲突
	machineID1 := int64(1)
	machineID2 := int64(2)
	
	gen1 := NewSnowflakeGenerator(&Options{MachineID: &machineID1})
	gen2 := NewSnowflakeGenerator(&Options{MachineID: &machineID2})
	
	ids := make(map[int64]bool)
	
	// 从两个生成器各生成1000个ID
	for i := 0; i < 1000; i++ {
		id1 := gen1.Generate()
		id2 := gen2.Generate()
		
		if ids[id1] {
			t.Errorf("生成了重复的ID: %d", id1)
		}
		if ids[id2] {
			t.Errorf("生成了重复的ID: %d", id2)
		}
		
		ids[id1] = true
		ids[id2] = true
	}
	
	if len(ids) != 2000 {
		t.Errorf("期望生成 2000 个唯一ID，但实际生成了 %d 个", len(ids))
	}
}

func BenchmarkSnowflakeGenerator_Generate(b *testing.B) {
	gen := NewSnowflakeGenerator(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkSnowflakeGenerator_GenerateConcurrent(b *testing.B) {
	gen := NewSnowflakeGenerator(nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gen.Generate()
		}
	})
}