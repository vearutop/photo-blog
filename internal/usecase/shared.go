package usecase

import (
	"io"
	"net/http"
)

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
