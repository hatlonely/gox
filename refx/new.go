package refx

import (
	"sync"
)

type constructor struct {
}

func newConstructor(newFunc any) (*constructor, error) {
	var constructor constructor

	return &constructor, nil
}

func (c *constructor) new(options any) (any, error) {
	return nil, nil
}

var nameConstructorMap sync.Map

func Register(namespace string, type_ string, constructor any) error {
	return nil
}

func New(namespace string, type_ string, options any) (any, error) {
	return nil, nil
}

func NewT[T any](options any) (T, error) {
	var t T
	return t, nil
}
