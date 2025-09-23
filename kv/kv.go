package kv

import (
	"github.com/hatlonely/gox/kv/store"
	"github.com/hatlonely/gox/ref"
)

func NewStoreWithOptions[K any, v any](options *ref.TypeOptions) (store.Store[K, v], error) {
	return NewStoreWithOptions[K, v](options)
}
