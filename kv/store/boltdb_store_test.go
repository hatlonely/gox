package store

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewBoltDBStoreWithOptions(t *testing.T) {
	Convey("TestNewBoltDBStoreWithOptions", t, func() {
		Convey("create boltdb store with default options", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath: dbPath,
			})
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()

			So(store.db, ShouldNotBeNil)
			So(store.keySerializer, ShouldNotBeNil)
			So(store.valSerializer, ShouldNotBeNil)
			So(store.dbPath, ShouldEqual, dbPath)
			So(string(store.bucketName), ShouldEqual, "default")
		})

		Convey("create boltdb store with custom serializers", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath: dbPath,
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
				BucketName: "test_bucket",
			})
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()

			So(string(store.bucketName), ShouldEqual, "test_bucket")
		})

		Convey("create boltdb store with snapshot type", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "zip",
			})
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()

			So(store.snapshotType, ShouldEqual, "zip")
		})
	})
}

func TestBoltDBStoreSet(t *testing.T) {
	Convey("TestBoltDBStoreSet", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("normal set", func() {
			err := store.Set(ctx, "key1", "value1")
			So(err, ShouldBeNil)

			val, err := store.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "value1")
		})

		Convey("set with if not exist - key not exist", func() {
			err := store.Set(ctx, "key2", "value2", WithIfNotExist())
			So(err, ShouldBeNil)

			val, err := store.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "value2")
		})

		Convey("set with if not exist - key exist", func() {
			err := store.Set(ctx, "key3", "value3")
			So(err, ShouldBeNil)

			err = store.Set(ctx, "key3", "new_value3", WithIfNotExist())
			So(err, ShouldEqual, ErrConditionFailed)

			val, err := store.Get(ctx, "key3")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "value3")
		})
	})
}

func TestBoltDBStoreGet(t *testing.T) {
	Convey("TestBoltDBStoreGet", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("get existing key", func() {
			err := store.Set(ctx, "existing_key", "existing_value")
			So(err, ShouldBeNil)

			val, err := store.Get(ctx, "existing_key")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "existing_value")
		})

		Convey("get non-existing key", func() {
			val, err := store.Get(ctx, "non_existing_key")
			So(err, ShouldEqual, ErrKeyNotFound)
			So(val, ShouldEqual, "")
		})

		Convey("get different data types", func() {
			intStore, err := NewBoltDBStoreWithOptions[string, int](&BoltDBStoreOptions{
				DBPath: dbPath + "_int",
			})
			So(err, ShouldBeNil)
			defer intStore.Close()

			err = intStore.Set(ctx, "int_key", 42)
			So(err, ShouldBeNil)

			val, err := intStore.Get(ctx, "int_key")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, 42)
		})

		Convey("get with slice type", func() {
			sliceStore, err := NewBoltDBStoreWithOptions[string, []string](&BoltDBStoreOptions{
				DBPath: dbPath + "_slice",
			})
			So(err, ShouldBeNil)
			defer sliceStore.Close()

			testData := []string{"item1", "item2", "item3"}
			err = sliceStore.Set(ctx, "slice_key", testData)
			So(err, ShouldBeNil)

			val, err := sliceStore.Get(ctx, "slice_key")
			So(err, ShouldBeNil)
			So(len(val), ShouldEqual, 3)
			So(val[0], ShouldEqual, "item1")
			So(val[1], ShouldEqual, "item2")
			So(val[2], ShouldEqual, "item3")
		})
	})
}

func TestBoltDBStoreDel(t *testing.T) {
	Convey("TestBoltDBStoreDel", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("delete existing key", func() {
			err := store.Set(ctx, "key_to_delete", "value")
			So(err, ShouldBeNil)

			err = store.Del(ctx, "key_to_delete")
			So(err, ShouldBeNil)

			val, err := store.Get(ctx, "key_to_delete")
			So(err, ShouldEqual, ErrKeyNotFound)
			So(val, ShouldEqual, "")
		})

		Convey("delete non-existing key", func() {
			err := store.Del(ctx, "non_existing_key")
			So(err, ShouldBeNil)
		})

		Convey("delete multiple keys", func() {
			keys := []string{"del_key1", "del_key2", "del_key3"}
			for i, key := range keys {
				err := store.Set(ctx, key, "value"+strconv.Itoa(i))
				So(err, ShouldBeNil)
			}

			for _, key := range keys {
				err := store.Del(ctx, key)
				So(err, ShouldBeNil)

				val, err := store.Get(ctx, key)
				So(err, ShouldEqual, ErrKeyNotFound)
				So(val, ShouldEqual, "")
			}
		})
	})
}

func TestBoltDBStoreBatchSet(t *testing.T) {
	Convey("TestBoltDBStoreBatchSet", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("batch set normal", func() {
			keys := []string{"batch_key1", "batch_key2", "batch_key3"}
			vals := []string{"batch_val1", "batch_val2", "batch_val3"}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, errItem := range errs {
				So(errItem, ShouldBeNil)
			}

			for i, key := range keys {
				val, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(val, ShouldEqual, vals[i])
			}
		})

		Convey("batch set with different length", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"val1", "val2", "val3"}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldNotBeNil)
			So(errs, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "keys and values length mismatch")
		})

		Convey("batch set with if not exist", func() {
			keys := []string{"ine_key1", "ine_key2", "ine_key3"}
			vals := []string{"ine_val1", "ine_val2", "ine_val3"}

			err := store.Set(ctx, "ine_key2", "existing_value")
			So(err, ShouldBeNil)

			errs, err := store.BatchSet(ctx, keys, vals, WithIfNotExist())
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			So(errs[0], ShouldBeNil)
			So(errs[1], ShouldEqual, ErrConditionFailed)
			So(errs[2], ShouldBeNil)

			val, err := store.Get(ctx, "ine_key1")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "ine_val1")

			val, err = store.Get(ctx, "ine_key2")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "existing_value")

			val, err = store.Get(ctx, "ine_key3")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "ine_val3")
		})
	})
}

func TestBoltDBStoreBatchGet(t *testing.T) {
	Convey("TestBoltDBStoreBatchGet", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		keys := []string{"bget_key1", "bget_key2", "bget_key3", "non_existing"}
		vals := []string{"bget_val1", "bget_val2", "bget_val3"}

		for i, key := range keys[:3] {
			err := store.Set(ctx, key, vals[i])
			So(err, ShouldBeNil)
		}

		Convey("batch get mixed existing and non-existing keys", func() {
			values, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(values), ShouldEqual, 4)
			So(len(errs), ShouldEqual, 4)

			So(values[0], ShouldEqual, "bget_val1")
			So(errs[0], ShouldBeNil)

			So(values[1], ShouldEqual, "bget_val2")
			So(errs[1], ShouldBeNil)

			So(values[2], ShouldEqual, "bget_val3")
			So(errs[2], ShouldBeNil)

			So(values[3], ShouldEqual, "")
			So(errs[3], ShouldEqual, ErrKeyNotFound)
		})
	})
}

func TestBoltDBStoreBatchDel(t *testing.T) {
	Convey("TestBoltDBStoreBatchDel", t, func() {
		dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
		defer os.RemoveAll(dbPath)

		store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
			DBPath: dbPath,
		})
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		keys := []string{"bdel_key1", "bdel_key2", "bdel_key3"}
		vals := []string{"bdel_val1", "bdel_val2", "bdel_val3"}

		for i, key := range keys {
			err := store.Set(ctx, key, vals[i])
			So(err, ShouldBeNil)
		}

		Convey("batch delete existing keys", func() {
			errs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, errItem := range errs {
				So(errItem, ShouldBeNil)
			}

			for _, key := range keys {
				val, err := store.Get(ctx, key)
				So(err, ShouldEqual, ErrKeyNotFound)
				So(val, ShouldEqual, "")
			}
		})

		Convey("batch delete non-existing keys", func() {
			nonExistingKeys := []string{"non1", "non2", "non3"}
			errs, err := store.BatchDel(ctx, nonExistingKeys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, errItem := range errs {
				So(errItem, ShouldBeNil)
			}
		})
	})
}

func TestBoltDBStoreClose(t *testing.T) {
	Convey("TestBoltDBStoreClose", t, func() {
		Convey("close normal store", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath: dbPath,
			})
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})

		Convey("close store with zip snapshot", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)
			defer os.Remove(dbPath + ".zip")

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "zip",
			})
			So(err, ShouldBeNil)

			ctx := context.Background()
			err = store.Set(ctx, "key", "value")
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})

		Convey("close store with tar.gz snapshot", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "tar.gz",
			})
			So(err, ShouldBeNil)

			ctx := context.Background()
			err = store.Set(ctx, "key", "value")
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})

		Convey("close store with invalid snapshot type", func() {
			dbPath := filepath.Join(os.TempDir(), "test_boltdb_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			defer os.RemoveAll(dbPath)

			store, err := NewBoltDBStoreWithOptions[string, string](&BoltDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "invalid",
			})
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported snapshot type")
		})
	})
}