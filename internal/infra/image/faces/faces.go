package faces

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/bool64/ctxd"
)

type Recognizer struct {
	logger ctxd.Logger
	c      *Client
	sem    chan struct{}
	cfg    RecognizerConfig
}

type RecognizerConfig struct {
	URL              string        `json:"url" example:"http://localhost:8011/"`
	Delay            time.Duration `json:"delay" description:"Cooldown delay between requests."` // Default tag is broken for time.Duration.
	ConcurrencyLimit int           `json:"concurrencyLimit" default:"1" description:"Max simultaneous requests."`
}

func NewRecognizer(logger ctxd.Logger, cfg RecognizerConfig) *Recognizer {
	r := &Recognizer{
		logger: logger,
	}

	if cfg.URL == "" {
		return r
	}

	r.c = NewClient(cfg.URL)
	r.cfg = cfg

	if cfg.ConcurrencyLimit <= 0 {
		cfg.ConcurrencyLimit = 1
	}

	r.sem = make(chan struct{}, cfg.ConcurrencyLimit)

	return r
}

func (r *Recognizer) RecognizeFile(ctx context.Context, fileName string) ([]GoFaceFace, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open file to get faces: %w", err)
	}

	return r.Recognize(ctx, f)
}

func (r *Recognizer) Recognize(ctx context.Context, jpg io.ReadCloser) (_ []GoFaceFace, err error) {
	defer func() {
		if err := jpg.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			r.logger.Error(ctx, "failed to close file to get faces", "error", err)
		}
	}()

	if r.c == nil {
		return nil, nil
	}

	if r.cfg.Delay > 0 {
		r.logger.Debug(ctx, "faces delay")
		time.Sleep(r.cfg.Delay)
	}

	r.sem <- struct{}{}
	defer func() {
		<-r.sem
	}()

	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	req := UploadImageRequest{}
	req.Image = UploadFile{
		Name:        "image.jpg",
		ContentType: "image/jpeg",
		Content:     jpg,
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
