package usecase

import (
	"context"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"io"
	"net/http"
	"time"
)

type hashInPath struct {
	Hash uniq.Hash `path:"hash"`
	req  *http.Request
}

func (s *hashInPath) SetRequest(r *http.Request) {
	s.req = r
}

type TextBody struct {
	t   string
	err error
}

func (t *TextBody) SetRequest(r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer func() {
		_ = r.Body.Close()
	}()

	t.t = string(b)
	t.err = err
}

func (t *TextBody) Text() (string, error) {
	return t.t, t.err
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
