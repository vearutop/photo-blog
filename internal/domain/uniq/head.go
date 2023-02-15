package uniq

import (
	"context"
	"time"
)

type Ensurer[V any] interface {
	Ensure(ctx context.Context, value V) (V, error)
}

type Finder[V any] interface {
	FindByHash(ctx context.Context, hash Hash) (V, error)
}

type Adder[V any] interface {
	Add(ctx context.Context, value V) error
}

type Updater[V any] interface {
	Update(ctx context.Context, value V) error
}

type Deleter[V any] interface {
	Delete(ctx context.Context, h Hash) error
}

type Time struct {
	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at"`
}

type Head struct {
	Time
	Hash Hash `db:"hash" json:"hash" description:"Unique hash value."`
}

func (h *Head) HashPtr() *Hash {
	return &h.Hash
}

func (h *Head) SetCreatedAt(t time.Time) {
	h.CreatedAt = t
}

func (h *Head) GetCreatedAt() time.Time {
	return h.CreatedAt
}
