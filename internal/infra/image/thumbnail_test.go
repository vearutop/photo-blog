package image

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bool64/ctxd"
	"github.com/nfnt/resize"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/ultrahdr"
)

func TestNoRotation(t *testing.T) {
	t.Skip()

	for _, fn := range []string{
		"./testdata/20240919_144111.jpg",
		"./testdata/20240919_144116.jpg",
		"./testdata/20240919_144120.jpg",
		"./testdata/20240919_144124.jpg",
	} {
		t.Run(fn, func(t *testing.T) {
			ctx := context.Background()

			img := photo.Image{}

			img.CreatedAt = time.Now()
			img.Path = fn

			_, err := ensureImageDimensions(ctx, &img, photo.IndexingFlags{})
			require.NoError(t, err)

			m, err := readMeta(ctx, &img)
			require.NoError(t, err)

			tt := NewThumbnailer(ctxd.NoOpLogger{})

			th, err := tt.Thumbnail(context.Background(), img, "300w")
			require.NoError(t, err)

			fmt.Println(fn, m.Rotate, th.Height, img.Width, img.Height)
		})
	}
}

func BenchmarkThumbnail_orig(b *testing.B) {
	sem := make(chan struct{}, 10)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()

			ctx := context.Background()
			img, err := loadJPEG(ctx, "./testdata/20240919_144111.jpg")
			require.NoError(b, err)

			buf := bytes.NewBuffer(nil)
			r := Resizer{
				Quality: 85,
				Interp:  resize.Lanczos2,
			}
			err = r.resizeJPEG(ctx, img, buf, 300, 200)
			if err != nil {
				b.Errorf("resizeJPEG failed: %v", err)
			}
		}()
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
}

func BenchmarkThumbnail_ult(b *testing.B) {
	sem := make(chan struct{}, 10)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sem <- struct{}{}

		go func() {
			defer func() { <-sem }()

			data, err := os.ReadFile("./testdata/20240919_144111.jpg")
			require.NoError(b, err)
			data, err = ultrahdr.ResizeJPEG(data, 300, 200, 85, ultrahdr.InterpolationLanczos2, false)

			if err != nil || len(data) == 0 {
				b.Errorf("resizeJPEG failed: %v", err)
			}
		}()
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
}
