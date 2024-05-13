package faces

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/bool64/ctxd"
)

type Recognizer struct {
	logger ctxd.Logger
	c      *Client
	mu     sync.Mutex
}

type RecognizerConfig struct {
	URL string `json:"url" example:"http://localhost:8011/"`
}

func NewRecognizer(logger ctxd.Logger, cfg RecognizerConfig) *Recognizer {
	r := &Recognizer{
		logger: logger,
	}

	if cfg.URL == "" {
		return r
	}

	r.c = NewClient(cfg.URL)

	return r
}

func (r *Recognizer) RecognizeFile(ctx context.Context, fileName string) ([]GoFaceFace, error) {
	if r.c == nil {
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open file to get faces: %w", err)
	}

	defer func() {
		if err := f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			r.logger.Error(ctx, "failed to close file to get faces", "error", err)
		}
	}()

	req := UploadImageRequest{}
	req.Image = UploadFile{
		Name:        path.Base(fileName),
		ContentType: "image/jpeg",
		Content:     f,
	}

	res, err := r.c.UploadImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get faces: %w", err)
	}

	r.logger.Info(ctx, "got faces", "faces", res.ValueOK.Found)

	faces := res.ValueOK.Faces
	if faces == nil {
		faces = []GoFaceFace{}
	}

	return faces, nil
}
