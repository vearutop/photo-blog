package uniq

import (
	"context"
	"time"

	"github.com/bool64/sqluct"
)

type EnsureOption[V any] struct {
	OnInsert func(st sqluct.StorageOf[V], o *sqluct.Options)
	OnUpdate func(st sqluct.StorageOf[V], o *sqluct.Options)
	Prepare  func(candidate *V, existing *V) (skipUpdate bool)
}

type Ensurer[V any] interface {
	Ensure(ctx context.Context, value V, options ...EnsureOption[V]) (V, error)
}

type Finder[V any] interface {
	FindByHash(ctx context.Context, hash Hash) (V, error)
	FindByHashes(ctx context.Context, hashes ...Hash) ([]V, error)
	Exists(ctx context.Context, hash Hash) (bool, error)
	FindAll(ctx context.Context) ([]V, error)
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
	CreatedAt time.Time `db:"created_at,omitempty" formType:"hidden" json:"created_at" title:"Created At" description:"Timestamp of creation."`
}

type Head struct {
	Time
	Hash Hash `db:"hash" formType:"hidden" json:"hash" description:"Unique hash value." title:"Hash Id"`
}

func (h *Head) HashPtr() *Hash {
	return &h.Hash
}

func (h *Head) CreatedAtPtr() *time.Time {
	return &h.CreatedAt
}

func (h *Head) SetCreatedAt(t time.Time) {
	h.CreatedAt = t
}

func (h *Head) GetCreatedAt() time.Time {
	return h.CreatedAt
}
