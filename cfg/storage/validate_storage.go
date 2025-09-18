package storage

import (
	"fmt"

	"github.com/hatlonely/gox/cfg/validator"
)

type ValidateStorage struct {
	storage Storage
}

func NewValidateStorage(storage Storage) *ValidateStorage {
	return &ValidateStorage{storage: storage}
}

func (vs *ValidateStorage) Sub(key string) Storage {
	if vs.storage == nil {
		return nil
	}
	return NewValidateStorage(vs.storage.Sub(key))
}

func (vs *ValidateStorage) ConvertTo(object interface{}) error {
	if vs.storage == nil {
		return nil
	}

	if err := vs.storage.ConvertTo(object); err != nil {
		return err
	}

	if err := validator.ValidateStruct(object); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	return nil
}

func (vs *ValidateStorage) Equals(other Storage) bool {
	if vs.storage == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	if o, ok := other.(*ValidateStorage); ok {
		return vs.storage.Equals(o.storage)
	}
	return vs.storage.Equals(other)
}
