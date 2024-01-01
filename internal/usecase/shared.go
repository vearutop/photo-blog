package usecase

import (
	"context"
	"time"

	"github.com/swaggest/rest/request"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type hashInPath struct {
	Hash uniq.Hash `path:"hash"`
	request.EmbeddedSetter
}

// detachedContext exposes parent values, but suppresses parent cancellation.
type detachedContext struct {
	parent context.Context //nolint:containedctx // This wrapping is here on purpose.
}

func (d detachedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (d detachedContext) Done() <-chan struct{} {
	return nil
}

func (d detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key interface{}) interface{} {
	return d.parent.Value(key)
}
