package store

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type LevelDBStoreOptions struct {
	// Source 是数据库文件的源路径。
	Source string `cfg:"source"`

	// 是否生成数据库路径后缀。
	GenerateDBPathSuffix bool `cfg:"generateDBPathSuffix"`

	// DBPath 是数据库文件的路径。
	// 不设置 GenerateDBPathSuffix 时，数据库将直接加载该目录，如果目录不存在，则自动创建并将数据写入到此路径。
	// 如果设置 GenerateDBPathSuffix, 以当前时间戳为后缀，创建新的目录。
	DBPath string `cfg:"dbPath" validate:"required"`

	// 快照类型。默认为空，不做快照。
	// 可选值：
	//   - zip: 使用 zip 格式压缩快照。
	//   - tar.gz: 使用 tar.gz 格式压缩快照。
	SnapshotType string `cfg:"snapshotType" validate:"omitempty,oneof=zip tar.gz"`

	// 键的序列化选项。
	KeySerializer *ref.TypeOptions `cfg:"keySerializer"`

	// 值的序列化选项。
	ValSerializer *ref.TypeOptions `cfg:"valSerializer"`

	// BlockCacher 提供 LevelDB 'sorted table' 块缓存的缓存算法。
	// 指定 NoCacher 以禁用缓存算法。
	//
	// 默认值是 LRUCacher。
	BlockCacher string `cfg:"blockCacher" validate:"omitempty,oneof=lru no"`

	// BlockCacheCapacity 定义 'sorted table' 块缓存的容量。
	// 使用 -1 表示零，这与指定 NoCacher 给 BlockCacher 具有相同的效果。
	//
	// 默认值是 8MiB。
	BlockCacheCapacity int `cfg:"blockCacheCapacity"`

	// BlockCacheEvictRemoved 允许在删除的 'sorted table' 上启用强制驱逐缓存块。
	//
	// 默认值是 false。
	BlockCacheEvictRemoved bool `cfg:"blockCacheEvictRemoved"`

	// BlockRestartInterval 是用于键的增量编码的重启点之间的键数。
	//
	// 默认值是 16。
	BlockRestartInterval int `cfg:"blockRestartInterval"`

	// BlockSize 是每个 'sorted table' 块的最小未压缩大小（以字节为单位）。
	//
	// 默认值是 4KiB。
	BlockSize int `cfg:"blockSize"`

	// CompactionExpandLimitFactor 限制压缩后扩展的大小。
	// 这将乘以压缩目标级别的表大小限制。
	//
	// 默认值是 25。
	CompactionExpandLimitFactor int `cfg:"compactionExpandLimitFactor"`

	// CompactionGPOverlapsFactor 限制单个 'sorted table' 生成的祖父（Level + 2）中的重叠。
	// 这将乘以祖父级别的表大小限制。
	//
	// 默认值是 10。
	CompactionGPOverlapsFactor int `cfg:"compactionGPOverlapsFactor"`

	// CompactionL0Trigger 定义触发压缩的 level-0 'sorted table' 数量。
	//
	// 默认值是 4。
	CompactionL0Trigger int `cfg:"compactionL0Trigger"`

	// CompactionSourceLimitFactor 限制压缩源大小。这不适用于 level-0。
	// 这将乘以压缩目标级别的表大小限制。
	//
	// 默认值是 1。
	CompactionSourceLimitFactor int `cfg:"compactionSourceLimitFactor"`

	// CompactionTableSize 限制压缩生成的 'sorted table' 大小。
	// 每个级别的限制将计算为：
	//   CompactionTableSize * (CompactionTableSizeMultiplier ^ Level)
	// 每个级别的乘数也可以使用 CompactionTableSizeMultiplierPerLevel 进行微调。
	//
	// 默认值是 2MiB。
	CompactionTableSize int `cfg:"compactionTableSize"`

	// CompactionTableSizeMultiplier 定义 CompactionTableSize 的乘数。
	//
	// 默认值是 1。
	CompactionTableSizeMultiplier float64 `cfg:"compactionTableSizeMultiplier"`

	// CompactionTableSizeMultiplierPerLevel 定义每级别的 CompactionTableSize 乘数。
	// 使用零跳过一个级别。
	//
	// 默认值是 nil。
	CompactionTableSizeMultiplierPerLevel []float64 `cfg:"compactionTableSizeMultiplierPerLevel"`

	// CompactionTotalSize 限制每个级别的 'sorted table' 总大小。
	// 每个级别的限制将计算为：
	//   CompactionTotalSize * (CompactionTotalSizeMultiplier ^ Level)
	// 每个级别的乘数也可以使用 CompactionTotalSizeMultiplierPerLevel 进行微调。
	//
	// 默认值是 10MiB。
	CompactionTotalSize int `cfg:"compactionTotalSize"`

	// CompactionTotalSizeMultiplier 定义 CompactionTotalSize 的乘数。
	//
	// 默认值是 10。
	CompactionTotalSizeMultiplier float64 `cfg:"compactionTotalSizeMultiplier"`

	// CompactionTotalSizeMultiplierPerLevel 定义每级别的 CompactionTotalSize 乘数。
	// 使用零跳过一个级别。
	//
	// 默认值是 nil。
	CompactionTotalSizeMultiplierPerLevel []float64 `cfg:"compactionTotalSizeMultiplierPerLevel"`

	// Compression 定义 'sorted table' 块压缩使用的压缩算法。
	//
	// 默认值（DefaultCompression）使用 snappy 压缩。
	Compression string `cfg:"compression" validate:"omitempty,oneof=default snappy none"`

	// DisableBufferPool 允许禁用 util.BufferPool 功能。
	//
	// 默认值是 false。
	DisableBufferPool bool `cfg:"disableBufferPool"`

	// DisableBlockCache 允许禁用 'sorted table' 块的 cache.Cache 功能。
	//
	// 默认值是 false。
	DisableBlockCache bool `cfg:"disableBlockCache"`

	// DisableCompactionBackoff 允许禁用压缩重试退避。
	//
	// 默认值是 false。
	DisableCompactionBackoff bool `cfg:"disableCompactionBackoff"`

	// DisableLargeBatchTransaction 允许禁用大批量写入时切换到事务模式。如果启用，大于 WriteBuffer 的批量写入将使用事务。
	//
	// 默认值是 false。
	DisableLargeBatchTransaction bool `cfg:"disableLargeBatchTransaction"`

	// ErrorIfExist 定义如果数据库已存在是否返回错误。
	//
	// 默认值是 false。
	ErrorIfExist bool `cfg:"errorIfExist"`

	// ErrorIfMissing 定义如果数据库丢失是否返回错误。如果为 false，则在丢失时将创建数据库，否则将返回错误。
	//
	// 默认值是 false。
	ErrorIfMissing bool `cfg:"errorIfMissing"`

	// IteratorSamplingRate 定义迭代器读取采样之间的近似间隔（以字节为单位）。样本将用于确定何时应触发压缩。
	//
	// 默认值是 1MiB。
	IteratorSamplingRate int `cfg:"iteratorSamplingRate"`

	// NoSync 允许完全禁用 fsync。
	//
	// 默认值是 false。
	NoSync bool `cfg:"noSync"`

	// NoWriteMerge 允许禁用写入合并。
	//
	// 默认值是 false。
	NoWriteMerge bool `cfg:"noWriteMerge"`

	// OpenFilesCacher 提供打开文件缓存的缓存算法。
	// 指定 NoCacher 以禁用缓存算法。
	//
	// 默认值是 LRUCacher。
	OpenFilesCacher string `cfg:"openFilesCacher" validate:"omitempty,oneof=lru no"`

	// OpenFilesCacheCapacity 定义打开文件缓存的容量。
	// 使用 -1 表示零，这与指定 NoCacher 给 OpenFilesCacher 具有相同的效果。
	//
	// 默认值是 500。
	OpenFilesCacheCapacity int `cfg:"openFilesCacheCapacity"`

	// 如果为 true，则以只读模式打开数据库。
	//
	// 默认值是 false。
	ReadOnly bool `cfg:"readOnly"`

	// Strict 定义数据库的严格级别。
	Strict string `cfg:"strict"`

	// WriteBuffer 定义 'memdb' 在刷新到 'sorted table' 之前的最大大小。'memdb' 是由磁盘上的未排序日志支持的内存数据库。
	//
	// LevelDB 可能同时持有最多两个 'memdb'。
	//
	// 默认值是 4MiB。
	WriteBuffer int `cfg:"writeBuffer"`

	// WriteL0StopTrigger 定义触发写入暂停的 level-0 'sorted table' 数量。
	//
	// 默认值是 12。
	WriteL0PauseTrigger int `cfg:"writeL0PauseTrigger"`

	// WriteL0SlowdownTrigger 定义触发写入减速的 level-0 'sorted table' 数量。
	//
	// 默认值是 8。
	WriteL0SlowdownTrigger int `cfg:"writeL0SlowdownTrigger"`
}

type LevelDBStore[K, V any] struct {
	db            *leveldb.DB
	keySerializer serializer.Serializer[K, []byte]
	valSerializer serializer.Serializer[V, []byte]

	dbPath       string
	snapshotType string
}

func NewLevelDBStoreWithOptions[K, V any](options *LevelDBStoreOptions) (*LevelDBStore[K, V], error) {
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

	blockCacher, err := leveldbParseCacher(options.BlockCacher)
	if err != nil {
		return nil, errors.WithMessage(err, "leveldbParseCacher failed")
	}
	compression, err := leveldbParseCompression(options.Compression)
	if err != nil {
		return nil, errors.WithMessage(err, "leveldbParseCompression failed")
	}
	strict, err := leveldbParseStrict(options.Strict)
	if err != nil {
		return nil, errors.WithMessage(err, "leveldbParseStrict failed")
	}
	openFilesCacher, err := leveldbParseCacher(options.OpenFilesCacher)
	if err != nil {
		return nil, errors.WithMessage(err, "leveldbParseCacher failed")
	}

	dbPath := options.DBPath
	if options.Source != "" || options.GenerateDBPathSuffix {
		dbPath = fmt.Sprintf("%s.%d", dbPath, time.Now().UnixNano())
	}
	if options.Source != "" {
		if err := extractTarGz(options.Source, dbPath); err != nil {
			return nil, errors.Wrapf(err, "extractTarGz failed. source: %s, dbPath: %s", options.Source, dbPath)
		}
	}

	db, err := leveldb.OpenFile(dbPath, &opt.Options{
		BlockCacher:                           blockCacher,
		BlockCacheCapacity:                    options.BlockCacheCapacity,
		BlockCacheEvictRemoved:                options.BlockCacheEvictRemoved,
		BlockRestartInterval:                  options.BlockRestartInterval,
		BlockSize:                             options.BlockSize,
		CompactionExpandLimitFactor:           options.CompactionExpandLimitFactor,
		CompactionGPOverlapsFactor:            options.CompactionGPOverlapsFactor,
		CompactionL0Trigger:                   options.CompactionL0Trigger,
		CompactionSourceLimitFactor:           options.CompactionSourceLimitFactor,
		CompactionTableSize:                   options.CompactionTableSize,
		CompactionTableSizeMultiplier:         options.CompactionTableSizeMultiplier,
		CompactionTableSizeMultiplierPerLevel: options.CompactionTableSizeMultiplierPerLevel,
		CompactionTotalSize:                   options.CompactionTotalSize,
		CompactionTotalSizeMultiplier:         options.CompactionTotalSizeMultiplier,
		CompactionTotalSizeMultiplierPerLevel: options.CompactionTotalSizeMultiplierPerLevel,
		Compression:                           compression,
		DisableBufferPool:                     options.DisableBufferPool,
		DisableBlockCache:                     options.DisableBlockCache,
		DisableCompactionBackoff:              options.DisableCompactionBackoff,
		DisableLargeBatchTransaction:          options.DisableLargeBatchTransaction,
		ErrorIfExist:                          options.ErrorIfExist,
		ErrorIfMissing:                        options.ErrorIfMissing,
		IteratorSamplingRate:                  options.IteratorSamplingRate,
		NoSync:                                options.NoSync,
		NoWriteMerge:                          options.NoWriteMerge,
		OpenFilesCacher:                       openFilesCacher,
		OpenFilesCacheCapacity:                options.OpenFilesCacheCapacity,
		ReadOnly:                              options.ReadOnly,
		Strict:                                strict,
		WriteBuffer:                           options.WriteBuffer,
		WriteL0PauseTrigger:                   options.WriteL0PauseTrigger,
		WriteL0SlowdownTrigger:                options.WriteL0SlowdownTrigger,
	})
	if err != nil {
		return nil, errors.Wrap(err, "leveldb.OpenFile failed. path: "+dbPath)
	}

	return &LevelDBStore[K, V]{
		db:            db,
		keySerializer: keySerializer,
		valSerializer: valSerializer,
		dbPath:        dbPath,
		snapshotType:  options.SnapshotType,
	}, nil
}

func leveldbParseStrict(strict string) (opt.Strict, error) {
	m := map[string]opt.Strict{
		"StrictManifest":        opt.StrictManifest,
		"StrictJournalChecksum": opt.StrictJournalChecksum,
		"StrictJournal":         opt.StrictJournal,
		"StrictBlockChecksum":   opt.StrictBlockChecksum,
		"StrictCompaction":      opt.StrictCompaction,
		"StrictReader":          opt.StrictReader,
		"StrictRecovery":        opt.StrictRecovery,
		"StrictOverride":        opt.StrictOverride,
		"StrictAll":             opt.StrictAll,
		"DefaultStrict":         opt.DefaultStrict,
		"NoStrict":              opt.NoStrict,

		"manifest":         opt.StrictManifest,
		"journal_checksum": opt.StrictJournalChecksum,
		"journal":          opt.StrictJournal,
		"block_checksum":   opt.StrictBlockChecksum,
		"compaction":       opt.StrictCompaction,
		"reader":           opt.StrictReader,
		"recovery":         opt.StrictRecovery,
		"override":         opt.StrictOverride,
		"all":              opt.StrictAll,
		"default":          opt.DefaultStrict,
		"none":             opt.NoStrict,
	}

	if strict == "" {
		return opt.DefaultStrict, nil
	}

	vals := strings.Split(strict, "|")
	var result opt.Strict
	for _, val := range vals {
		v, ok := m[val]
		if !ok {
			return 0, errors.Errorf("invalid strict value: %s", val)
		}

		result |= v
	}

	return result, nil
}

func leveldbParseCompression(compression string) (opt.Compression, error) {
	m := map[string]opt.Compression{
		"DefaultCompression": opt.DefaultCompression,
		"NoCompression":      opt.NoCompression,
		"SnappyCompression":  opt.SnappyCompression,

		"default": opt.DefaultCompression,
		"none":    opt.NoCompression,
		"snappy":  opt.SnappyCompression,
	}

	if compression == "" {
		return opt.DefaultCompression, nil
	}

	val, ok := m[compression]
	if !ok {
		return 0, errors.Errorf("invalid compression value: %s", compression)
	}

	return val, nil
}

func leveldbParseCacher(cacher string) (opt.Cacher, error) {
	m := map[string]opt.Cacher{
		"LRUCacher": opt.LRUCacher,
		"NoCacher":  opt.NoCacher,

		"lru":  opt.LRUCacher,
		"none": opt.NoCacher,
	}

	if cacher == "" {
		return opt.DefaultBlockCacher, nil
	}

	val, ok := m[cacher]
	if !ok {
		return nil, errors.Errorf("invalid cacher value: %s", cacher)
	}

	return val, nil
}
