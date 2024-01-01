package sqlitec

import (
	"context"
	"database/sql"

	"github.com/bool64/cache"
)

var (
	_ cache.ReadWriter = &CacheBackend{}
	_ cache.Deleter    = &CacheBackend{}
)

type CacheBackend struct {
	sql.DB
}

func (c *CacheBackend) Delete(ctx context.Context, key []byte) error {
	// TODO implement me
	panic("implement me")
}

func (c *CacheBackend) Read(ctx context.Context, key []byte) (interface{}, error) {
	// TODO implement me
	panic("implement me")
}

func (c *CacheBackend) Write(ctx context.Context, key []byte, value interface{}) error {
	// TODO implement me
	panic("implement me")
}
