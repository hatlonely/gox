package store

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cockroachdb/fifo"
	"github.com/cockroachdb/pebble"
	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type PebbleStoreOptions struct {
	// Source 是数据库文件的源路径。
	Source string

	// 是否生成数据库路径后缀。
	GenerateDBPathSuffix bool

	// DBPath 是数据库文件的路径。
	// 不设置 GenerateDBPathSuffix 时，数据库将直接加载该目录，如果目录不存在，则自动创建并将数据写入到此路径。
	// 如果设置 GenerateDBPathSuffix, 以当前时间戳为后缀，创建新的目录。
	DBPath string `validate:"required"`

	// 快照类型。默认为空，不做快照。
	// 可选值：
	//   - zip: 使用 zip 格式压缩快照。
	//   - tar.gz: 使用 tar.gz 格式压缩快照。
	SnapshotType string

	// 指定是否在写入时同步到磁盘。
	SetWithoutSync bool

	// 键的序列化选项。
	KeySerializer *ref.TypeOptions

	// 值的序列化选项。
	ValSerializer *ref.TypeOptions

	// Sync sstables 定期同步以平滑写入磁盘的过程。
	// 此选项不提供任何持久性保证，但用于避免操作系统自动决定写入大量脏文件系统缓冲区时的延迟峰值。
	// 此选项仅控制 SSTable 同步；WAL 同步由 WALBytesPerSync 控制。
	//
	// 默认值为 512KB。
	BytesPerSync int

	// Cache 用于缓存来自 sstables 的未压缩块。
	//
	// 默认缓存大小为 8 MB。
	Cache *struct {
		Size int64
	}

	// LoadBlockSema，如果设置，用于限制可以并行加载（即从文件系统读取）的块数。
	// 每次加载在读取期间从信号量中获取一个单位。
	LoadBlockSema *struct {
		Capacity int64
	}

	// DisableWAL 禁用预写日志（WAL）。
	// 禁用预写日志禁止崩溃恢复，但如果不需要崩溃恢复（例如，仅在数据库中存储临时状态），则可以提高性能。
	//
	// TODO：未测试
	DisableWAL bool

	// ErrorIfExists 如果数据库已存在，则在 Open 时引发错误。
	// 可以使用 errors.Is(err, ErrDBAlreadyExists) 检查错误。
	//
	// 默认值为 false。
	ErrorIfExists bool

	// ErrorIfNotExists 如果数据库不存在，则在 Open 时引发错误。
	// 可以使用 errors.Is(err, ErrDBDoesNotExist) 检查错误。
	//
	// 默认值为 false，这将导致在数据库不存在时创建数据库。
	ErrorIfNotExists bool

	// ErrorIfNotPristine 如果数据库已存在并且已对数据库执行任何操作，则在 Open 时引发错误。
	// 可以使用 errors.Is(err, ErrDBNotPristine) 检查错误。
	//
	// 请注意，包含所有已删除键的数据库可能会或可能不会触发错误。
	// 目前，我们检查是否有任何活动的 SST 或日志记录需要重放。
	ErrorIfNotPristine bool

	// Experimental 包含默认关闭的实验选项。
	// 这些选项是临时的，最终将被删除、移出实验组或成为不可调整的默认值。
	// 这些选项可能随时更改，因此不要依赖它们。
	Experimental struct {
		// 启用压缩并发的 L0 读放大阈值（如果未超过 CompactionDebtConcurrency）。
		// 每个此值的倍数启用另一个并发压缩，最多达到 MaxConcurrentCompactions。
		L0CompactionConcurrency int

		// CompactionDebtConcurrency 控制压缩债务的阈值，在该阈值下添加额外的压缩并发槽。
		// 每个此值的倍数在压缩债务字节中添加一个额外的并发压缩。
		// 这在 L0CompactionConcurrency 之上工作，因此选择由两个选项确定的压缩并发槽的较高计数。
		CompactionDebtConcurrency uint64

		// ReadCompactionRate 通过调整 manifest.FileMetadata 中的 `AllowedSeeks` 控制读取触发的压缩频率：
		//
		// AllowedSeeks = FileSize / ReadCompactionRate
		//
		// 来自 LevelDB：
		// ```
		// 我们安排在一定数量的查找后自动压缩此文件。假设：
		//   (1) 一次查找花费 10ms
		//   (2) 写入或读取 1MB 花费 10ms（100MB/s）
		//   (3) 1MB 的压缩涉及 25MB 的 IO：
		//         从此级别读取 1MB
		//         从下一级读取 10-12MB（边界可能未对齐）
		//         写入下一级 10-12MB
		// 这意味着 25 次查找的成本与压缩 1MB 数据的成本相同。即，一次查找的成本大约等于压缩 40KB 数据的成本。
		// 我们有点保守，允许大约每 16KB 数据进行一次查找，然后触发压缩。
		// ```
		ReadCompactionRate int64

		// ReadSamplingMultiplier 是 iterator.maybeSampleRead() 中 readSamplingPeriod 的乘数，用于控制读取采样的频率以触发读取触发的压缩。
		// 值为 -1 时禁止采样并禁用读取触发的压缩。默认值为 1 << 4，与常数 1 << 16 相乘得到 1 << 20（1MB）。
		ReadSamplingMultiplier int64

		// TableCacheShards 是每个表缓存的分片数。
		// 减少该值可以减少每个 DB 实例的空闲 goroutine 数量，这在具有大量 DB 实例和大量 CPU 的场景中很有用，但这样做可能会导致表缓存中的争用增加和性能下降。
		//
		// 默认值为逻辑 CPU 数量，可以通过 runtime.GOMAXPROCS 限制。
		TableCacheShards int

		// ValidateOnIngest 在 sstables 被引入后安排验证。
		//
		// 默认情况下，此值为 false。
		ValidateOnIngest bool

		// LevelMultiplier 配置用于确定 LSM 每个级别所需大小的大小乘数。默认值为 10。
		LevelMultiplier int

		// MaxWriterConcurrency 用于指示压缩队列允许使用的最大压缩工作者数量。
		// 如果 MaxWriterConcurrency > 0，则 Writer 将使用并行性来压缩和写入块到磁盘。否则，Writer 将同步压缩和写入块到磁盘。
		MaxWriterConcurrency int

		// ForceWriterParallelism 用于在变形测试中强制 sstable Writer 的并行性。
		// 即使设置了 MaxWriterConcurrency 选项，我们也仅在有足够的 CPU 可用时启用 sstable Writer 中的并行性，此选项绕过该限制。
		ForceWriterParallelism bool

		// CacheSizeBytesBytes 是共享存储上对象的磁盘块缓存大小（以字节为单位）。
		// 如果为 0，则不使用缓存。
		SecondaryCacheSizeBytes int64
	}

	// FlushDelayDeleteRange 配置数据库在强制刷新包含范围删除的 memtable 之前应等待的时间。
	// 只有在刷新范围删除后才能回收磁盘空间。如果为零，则不会自动刷新。
	FlushDelayDeleteRange time.Duration

	// FlushDelayRangeKey 配置数据库在强制刷新包含范围键的 memtable 之前应等待的时间。
	// memtable 中的范围键会阻止懒惰的组合迭代，因此希望尽快刷新范围键。如果为零，则不会自动刷新。
	FlushDelayRangeKey time.Duration

	// FlushSplitBytes 表示每个刷新拆分间隔（即两个刷新拆分键之间的范围）中每个子级别的目标字节数。
	// 当设置为零时，每次刷新仅生成一个 sstable。当设置为非零值时，刷新会在点处拆分，以满足 L0 的 TargetFileSize、任何与祖父母相关的重叠选项以及 L0 刷新拆分间隔的边界键（目标是在每对边界键之间的每个子级别中包含大约 FlushSplitBytes 字节）。
	// 在刷新期间拆分 sstables 允许在将这些表压缩到较低级别时增加压缩灵活性和并发性。
	FlushSplitBytes int64

	// 触发 L0 压缩所需的 L0 文件数量。
	L0CompactionFileThreshold int

	// 触发 L0 压缩所需的 L0 读放大数量。
	L0CompactionThreshold int

	// L0 读放大的硬限制，计算为 L0 子级别的数量。
	// 当达到此阈值时，写入将停止。
	L0StopWritesThreshold int

	// LBase 的最大字节数。基级是 L0 被压缩到的级别。
	// 基级是根据 LSM 中现有数据动态确定的。其他级别的最大字节数是根据基级的最大大小动态计算的。
	// 当级别的最大字节数超过时，请求压缩。
	LBaseMaxBytes int64

	// MaxManifestFileSize 是 MANIFEST 文件允许的最大大小。
	// 当 MANIFEST 超过此大小时，它将被滚动并创建一个新的 MANIFEST。
	MaxManifestFileSize int64

	// MaxOpenFiles 是可以由 DB 使用的打开文件的软限制。
	//
	// 默认值为 1000。
	MaxOpenFiles int

	// 稳态下 MemTable 的大小。实际的 MemTable 大小从 min(256KB, MemTableSize) 开始，并为每个后续 MemTable 翻倍，直到达到 MemTableSize。
	// 这减少了短命（测试）DB 实例的 MemTable 引起的内存压力。
	// 请注意，由于刷新 MemTable 涉及创建一个新 MemTable 并在后台写入旧 MemTable 的内容，因此可以存在多个 MemTable。
	// MemTableStopWritesThreshold 对排队的 MemTable 大小设置了硬限制。
	//
	// 默认值为 4MB。
	MemTableSize int

	// 排队的 MemTable 数量的硬限制。
	// 当排队的 MemTable 大小总和超过：MemTableStopWritesThreshold * MemTableSize 时，写入将停止。
	//
	// 此值应至少为 2，否则每当 MemTable 正在刷新时写入将停止。
	//
	// 默认值为 2。
	MemTableStopWritesThreshold int

	// DisableAutomaticCompactions 指定是否调度自动压缩。默认值为 false（启用）。
	// 此选项仅在运行手动压缩时外部使用，内部用于测试。
	DisableAutomaticCompactions bool

	// NoSyncOnClose 决定 Pebble 实例是否会强制对其写入的文件进行关闭时同步（例如，fdatasync() 或 sync_file_range()）。
	// 将此设置为 true 会删除关闭时同步的保证。一些实现仍然可以发出非阻塞同步。
	NoSyncOnClose bool

	// NumPrevManifest 是我们希望保留用于调试目的的非当前或旧的清单数量。
	// 默认情况下，我们将保留一个旧的清单。
	NumPrevManifest int

	// ReadOnly 表示应以只读模式打开数据库。
	// 对数据库的写入将返回错误，禁用后台压缩，并且在启动时重放 WAL 后通常发生的刷新被禁用。
	ReadOnly bool

	// WALBytesPerSync 设置在后台调用 Sync 之前写入 WAL 的字节数。
	// 就像上面的 BytesPerSync 一样，这有助于平滑磁盘写入延迟，并避免操作系统一次写入大量缓冲数据的情况。
	// 但是，这对于 WAL 来说不太必要，因为许多写入操作已经传递了 Sync = true。
	//
	// 默认值为 0，即无后台同步。这与 RocksDB 中的默认行为相匹配。
	WALBytesPerSync int

	// WALDir 指定存储预写日志（WAL）的目录。如果为空（默认），WAL 将存储在与 sstables 相同的目录中（即传递给 pebble.Open 的目录）。
	WALDir string

	// TargetByteDeletionRate 是限制 sstable 文件删除的速率（以每秒字节数为单位）。
	//
	// 删除节奏用于在压缩完成或读取器关闭并且新废弃的文件需要清理时减慢删除速度。
	// 一次删除大量文件可能会导致某些 SSD 上的磁盘延迟增加，此功能可以防止这种情况。
	//
	// 此值仅是一个尽力而为的目标；如果删除落后或磁盘空间不足，有效速率可能会更高。
	//
	// 将此设置为 0 禁用删除节奏，这也是默认值。
	TargetByteDeletionRate int
}

type PebbleStore[K, V any] struct {
	db            *pebble.DB
	keyMarshaller serializer.Serializer[K, []byte]
	valMarshaller serializer.Serializer[V, []byte]
	setOptions    *pebble.WriteOptions

	dbPath       string
	snapshotType string
}

func NewPebbleStoreWithOptions[K, V any](options *PebbleStoreOptions) (*PebbleStore[K, V], error) {
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
	valueSerializer, ok := valSerializerInterface.(serializer.Serializer[V, []byte])
	if !ok {
		return nil, errors.New("invalid value serializer type")
	}

	pebbleOptions := &pebble.Options{
		BytesPerSync:                options.BytesPerSync,
		DisableWAL:                  options.DisableWAL,
		ErrorIfExists:               options.ErrorIfExists,
		ErrorIfNotExists:            options.ErrorIfNotExists,
		ErrorIfNotPristine:          options.ErrorIfNotPristine,
		FlushDelayDeleteRange:       options.FlushDelayDeleteRange,
		FlushDelayRangeKey:          options.FlushDelayRangeKey,
		FlushSplitBytes:             options.FlushSplitBytes,
		L0CompactionFileThreshold:   options.L0CompactionFileThreshold,
		L0CompactionThreshold:       options.L0CompactionThreshold,
		L0StopWritesThreshold:       options.L0StopWritesThreshold,
		LBaseMaxBytes:               options.LBaseMaxBytes,
		MaxManifestFileSize:         options.MaxManifestFileSize,
		MaxOpenFiles:                options.MaxOpenFiles,
		MemTableSize:                uint64(options.MemTableSize),
		MemTableStopWritesThreshold: options.MemTableStopWritesThreshold,
		DisableAutomaticCompactions: options.DisableAutomaticCompactions,
		NoSyncOnClose:               options.NoSyncOnClose,
		NumPrevManifest:             options.NumPrevManifest,
		ReadOnly:                    options.ReadOnly,
		WALBytesPerSync:             options.WALBytesPerSync,
		WALDir:                      options.WALDir,
		TargetByteDeletionRate:      options.TargetByteDeletionRate,
	}
	pebbleOptions.Experimental.L0CompactionConcurrency = options.Experimental.L0CompactionConcurrency
	pebbleOptions.Experimental.CompactionDebtConcurrency = options.Experimental.CompactionDebtConcurrency
	pebbleOptions.Experimental.ReadCompactionRate = options.Experimental.ReadCompactionRate
	pebbleOptions.Experimental.ReadSamplingMultiplier = options.Experimental.ReadSamplingMultiplier
	pebbleOptions.Experimental.TableCacheShards = options.Experimental.TableCacheShards
	pebbleOptions.Experimental.ValidateOnIngest = options.Experimental.ValidateOnIngest
	pebbleOptions.Experimental.LevelMultiplier = options.Experimental.LevelMultiplier
	pebbleOptions.Experimental.MaxWriterConcurrency = options.Experimental.MaxWriterConcurrency
	pebbleOptions.Experimental.ForceWriterParallelism = options.Experimental.ForceWriterParallelism
	pebbleOptions.Experimental.SecondaryCacheSizeBytes = options.Experimental.SecondaryCacheSizeBytes
	if options.Cache != nil {
		pebbleOptions.Cache = pebble.NewCache(options.Cache.Size)
	}
	if options.LoadBlockSema != nil {
		pebbleOptions.LoadBlockSema = fifo.NewSemaphore(options.LoadBlockSema.Capacity)
	}

	//sst表压缩级别，默认是 SnappyCompression
	//经本地测试使用默认级别 + tar.gz 压缩后的包最小，比 Zstd + tar.gz 还小
	//pebbleOptions.Levels = make([]pebble.LevelOptions, 7)
	//for i := range pebbleOptions.Levels {
	//	pebbleOptions.Levels[i].Compression = pebble.ZstdCompression
	//}

	dbPath := options.DBPath
	if options.Source != "" || options.GenerateDBPathSuffix {
		dbPath = fmt.Sprintf("%s.%d", dbPath, time.Now().UnixNano())
	}
	if options.Source != "" {
		if strings.HasSuffix(options.Source, ".tar.gz") {
			if err = extractTarGz(options.Source, dbPath); err != nil {
				return nil, errors.Wrapf(err, "extractTarGz failed. source: %s, dbPath: %s", options.Source, dbPath)
			}
		} else if strings.HasSuffix(options.Source, ".zip") {
			if err = extractZip(options.Source, dbPath); err != nil {
				return nil, errors.Wrapf(err, "extractZip failed. source: %s, dbPath: %s", options.Source, dbPath)
			}
		} else {
			return nil, errors.Errorf("unsupported source file type. source: %s", options.Source)
		}
	}

	// 创建 Pebble 数据库
	db, err := pebble.Open(dbPath, pebbleOptions)
	if err != nil {
		return nil, errors.Wrap(err, "pebble.Open failed")
	}

	setOptions := pebble.Sync
	if options.SetWithoutSync {
		setOptions = pebble.NoSync
	}

	// 创建 PebbleStore 实例
	return &PebbleStore[K, V]{
		db:            db,
		keyMarshaller: keySerializer,
		valMarshaller: valueSerializer,
		setOptions:    setOptions,
		dbPath:        dbPath,
		snapshotType:  options.SnapshotType,
	}, nil
}
