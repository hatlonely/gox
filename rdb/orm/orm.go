package orm

import "context"

type ORM[T any] interface {
	Create(ctx context.Context, record T) error
}
