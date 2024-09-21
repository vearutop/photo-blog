package image

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bool64/ctxd"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/photo"
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
