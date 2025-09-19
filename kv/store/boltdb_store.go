package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type BoltDBStoreOptions struct {
	// Source 是数据库文件的源路径。
	Source string `cfg:"source"`

	// 是否生成数据库路径后缀。
	GenerateDBPathSuffix bool `cfg:"generateDBPathSuffix"`

	// DBPath 是数据库文件的路径。
	// 如果 Source 为空，数据库将直接加载该文件，如果该文件不存在，则自动创建并将数据写入到此路径。
	// 如果 Source 不为空, 则数据库将复制 Source 文件到 DBPath，并以当前时间戳为后缀，然后加载该文件。
	DBPath string `cfg:"dbPath" validate:"required"`

	// 键的序列化选项。
	KeySerializer *ref.TypeOptions `cfg:"keySerializer"`

	// 值的序列化选项。
	ValSerializer *ref.TypeOptions `cfg:"valSerializer"`

	// Timeout 是获取文件锁的等待时间。
	// 设置为零时将无限期等待。此选项仅在 Darwin 和 Linux 上可用。
	Timeout time.Duration `cfg:"timeout"`

	// 在内存映射文件之前设置 DB.NoGrowSync 标志。
	NoGrowSync bool `cfg:"noGrowSync"`

	// 不将 freelist 同步到磁盘。这在正常操作下提高了数据库写入性能，
	// 但在恢复期间需要完全重新同步数据库。
	NoFreelistSync bool `cfg:"noFreelistSync"`

	// FreelistType 设置后端 freelist 类型。有两种选择：
	// array 简单但如果数据库很大且 freelist 中的碎片常见，性能会急剧下降。
	// 另一种选择是使用 hashmap，它在几乎所有情况下都更快，
	// 但不能保证提供最小的可用页面 ID。在正常情况下是安全的。
	// 默认类型是 array
	FreelistType string `cfg:"freelistType" validate:"omitempty,oneof=array hashmap"`

	// 以只读模式打开数据库。使用 flock(..., LOCK_SH |LOCK_NB) 获取共享锁（UNIX）。
	ReadOnly bool `cfg:"readOnly"`

	// 在内存映射文件之前设置 DB.MmapFlags 标志。
	MmapFlags int `cfg:"mmapFlags"`

	// InitialMmapSize 是数据库的初始 mmap 大小（以字节为单位）。
	// 如果 InitialMmapSize 足够大以容纳数据库 mmap 大小，则读事务不会阻塞写事务。
	// （有关更多信息，请参见 DB.Begin）
	//
	// 如果 <=0，则初始映射大小为 0。
	// 如果 initialMmapSize 小于之前的数据库大小，则不起作用。
	InitialMmapSize int `validate:"min=0"`

	// PageSize 覆盖默认的操作系统页面大小。
	PageSize int `cfg:"pageSize"`

	// NoSync 设置 DB.NoSync 的初始值。通常可以直接在从 Open() 返回的 DB 上设置，
	// 但此选项在暴露 Options 而不是底层 DB 的 API 中很有用。
	NoSync bool `cfg:"noSync"`

	// 快照类型。默认为空，不做快照。
	// 可选值：
	//   - zip: 使用 zip 格式压缩快照。
	//   - tar.gz: 使用 tar.gz 格式压缩快照。
	SnapshotType string `cfg:"snapshotType" validate:"omitempty,oneof=zip tar.gz"`

	// 默认桶名称
	BucketName string `cfg:"bucketName"`
}

type BoltDBStore[K, V any] struct {
	db            *bolt.DB
	keySerializer serializer.Serializer[K, []byte]
	valSerializer serializer.Serializer[V, []byte]

	dbPath       string
	snapshotType string
	bucketName   []byte
}

func NewBoltDBStoreWithOptions[K, V any](options *BoltDBStoreOptions) (*BoltDBStore[K, V], error) {
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

	dbPath := options.DBPath
	directory := filepath.Dir(dbPath)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, errors.Wrapf(err, "os.MkdirAll failed. directory: %s", directory)
	}
	if options.Source != "" || options.GenerateDBPathSuffix {
		dbPath = fmt.Sprintf("%s.%d", dbPath, time.Now().UnixNano())
	}
	if options.Source != "" {
		if strings.HasSuffix(options.Source, ".tar.gz") {
			if err := extractTarGz(options.Source, dbPath); err != nil {
				return nil, errors.Wrapf(err, "extractTarGz failed. source: %s, dbPath: %s", options.Source, dbPath)
			}
		} else if strings.HasSuffix(options.Source, ".zip") {
			if err := extractZip(options.Source, dbPath); err != nil {
				return nil, errors.Wrapf(err, "extractZip failed. source: %s, dbPath: %s", options.Source, dbPath)
			}
		} else {
			// 直接复制文件
			srcFile, err := os.Open(options.Source)
			if err != nil {
				return nil, errors.Wrapf(err, "os.Open failed. source: %s", options.Source)
			}

			dstFile, err := os.Create(dbPath)
			if err != nil {
				return nil, errors.Wrapf(err, "os.Create failed. dbPath: %s", dbPath)
			}

			if _, err = io.Copy(dstFile, srcFile); err != nil {
				return nil, errors.Wrap(err, "io.Copy failed")
			}

			if err = dstFile.Sync(); err != nil {
				return nil, errors.Wrap(err, "dstFile.Sync failed")
			}

			if err := srcFile.Close(); err != nil {
				return nil, errors.Wrap(err, "srcFile.Close failed")
			}
			if err := dstFile.Close(); err != nil {
				return nil, errors.Wrap(err, "dstFile.Close failed")
			}
		}
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout:         options.Timeout,
		NoGrowSync:      options.NoGrowSync,
		NoFreelistSync:  options.NoFreelistSync,
		FreelistType:    bolt.FreelistType(options.FreelistType),
		ReadOnly:        options.ReadOnly,
		MmapFlags:       options.MmapFlags,
		InitialMmapSize: options.InitialMmapSize,
		PageSize:        options.PageSize,
		NoSync:          options.NoSync,
	})
	if err != nil {
		return nil, err
	}

	// 设置默认桶名称
	bucketName := "default"
	if options.BucketName != "" {
		bucketName = options.BucketName
	}

	store := &BoltDBStore[K, V]{
		db:            db,
		keySerializer: keySerializer,
		valSerializer: valSerializer,
		dbPath:        dbPath,
		snapshotType:  options.SnapshotType,
		bucketName:    []byte(bucketName),
	}

	// 创建默认桶
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(store.bucketName)
		return err
	})
	if err != nil {
		return nil, errors.Wrap(err, "create bucket failed")
	}

	return store, nil
}

func (s *BoltDBStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return errors.Wrap(err, "marshal key failed")
	}

	valueBytes, err := s.valSerializer.Serialize(value)
	if err != nil {
		return errors.Wrap(err, "marshal value failed")
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return errors.New("bucket not found")
		}

		if options.IfNotExist {
			existing := bucket.Get(keyBytes)
			if existing != nil {
				return ErrConditionFailed
			}
		}

		return bucket.Put(keyBytes, valueBytes)
	})
}

func (s *BoltDBStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zeroV V

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return zeroV, errors.Wrap(err, "marshal key failed")
	}

	var valueBytes []byte
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return errors.New("bucket not found")
		}

		data := bucket.Get(keyBytes)
		if data == nil {
			return ErrKeyNotFound
		}

		// 复制数据，因为 BoltDB 会重用内存
		valueBytes = make([]byte, len(data))
		copy(valueBytes, data)
		return nil
	})

	if err != nil {
		return zeroV, err
	}

	value, err := s.valSerializer.Deserialize(valueBytes)
	if err != nil {
		return zeroV, errors.Wrap(err, "unmarshal value failed")
	}

	return value, nil
}

func (s *BoltDBStore[K, V]) Del(ctx context.Context, key K) error {
	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return errors.Wrap(err, "marshal key failed")
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return errors.New("bucket not found")
		}

		return bucket.Delete(keyBytes)
	})
}

func (s *BoltDBStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, errors.New("keys and values length mismatch")
	}

	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	errs := make([]error, len(keys))

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return errors.New("bucket not found")
		}

		for i, key := range keys {
			keyBytes, err := s.keySerializer.Serialize(key)
			if err != nil {
				errs[i] = errors.Wrap(err, "marshal key failed")
				continue
			}

			valueBytes, err := s.valSerializer.Serialize(vals[i])
			if err != nil {
				errs[i] = errors.Wrap(err, "marshal value failed")
				continue
			}

			if options.IfNotExist {
				existing := bucket.Get(keyBytes)
				if existing != nil {
					errs[i] = ErrConditionFailed
					continue
				}
			}

			if err := bucket.Put(keyBytes, valueBytes); err != nil {
				errs[i] = errors.Wrap(err, "put failed")
			}
		}

		return nil
	})

	return errs, err
}

func (s *BoltDBStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	vals := make([]V, len(keys))
	errs := make([]error, len(keys))

	for i, key := range keys {
		val, err := s.Get(ctx, key)
		vals[i] = val
		errs[i] = err
	}

	return vals, errs, nil
}

func (s *BoltDBStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errs := make([]error, len(keys))

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return errors.New("bucket not found")
		}

		for i, key := range keys {
			keyBytes, err := s.keySerializer.Serialize(key)
			if err != nil {
				errs[i] = errors.Wrap(err, "marshal key failed")
				continue
			}

			if err := bucket.Delete(keyBytes); err != nil {
				errs[i] = errors.Wrap(err, "delete failed")
			}
		}

		return nil
	})

	return errs, err
}

func (s *BoltDBStore[K, V]) Close() error {
	if s.db == nil {
		return nil
	}

	// 先关闭数据库
	if err := s.db.Close(); err != nil {
		return errors.Wrap(err, "close database failed")
	}

	// 标记数据库已关闭，防止重复关闭
	s.db = nil

	// 如果设置了快照类型，制作快照
	if s.snapshotType != "" {
		snapshotPath := fmt.Sprintf("%s.%d.%s", s.dbPath, time.Now().UnixNano(), s.snapshotType)

		switch s.snapshotType {
		case "zip":
			if err := createZip(s.dbPath, snapshotPath); err != nil {
				return errors.Wrapf(err, "create zip snapshot failed. dbPath: %s, snapshotPath: %s", s.dbPath, snapshotPath)
			}
		case "tar.gz":
			if err := createTarGz(s.dbPath, snapshotPath); err != nil {
				return errors.Wrapf(err, "create tar.gz snapshot failed. dbPath: %s, snapshotPath: %s", s.dbPath, snapshotPath)
			}
		default:
			return errors.Errorf("unsupported snapshot type: %s", s.snapshotType)
		}
	}

	return nil
}

var _ Store[string, string] = (*BoltDBStore[string, string])(nil)
