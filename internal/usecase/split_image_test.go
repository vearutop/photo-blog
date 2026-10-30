package usecase

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteImageSplitZIP(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 7, 5))
	for y := range 5 {
		for x := range 7 {
			src.Set(x, y, color.RGBA{R: uint8(x * 30), G: uint8(y * 40), B: 100, A: 255})
		}
	}

	var archive bytes.Buffer
	require.NoError(t, writeImageSplitZIP(&archive, src, 3))

	zr, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, 9)

	for _, tc := range []struct {
		name          string
		width, height int
	}{
		{"row-01-col-01.jpg", 2, 1},
		{"row-02-col-02.jpg", 2, 2},
		{"row-03-col-03.jpg", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var file *zip.File
			for _, f := range zr.File {
				if f.Name == tc.name {
					file = f
					break
				}
			}
			require.NotNil(t, file)

			r, err := file.Open()
			require.NoError(t, err)
			defer r.Close()

			cfg, err := jpeg.DecodeConfig(r)
			require.NoError(t, err)
			require.Equal(t, tc.width, cfg.Width)
			require.Equal(t, tc.height, cfg.Height)
		})
	}
}

func TestWriteImageSplitZIPRejectsZeroGrid(t *testing.T) {
	t.Parallel()

	err := writeImageSplitZIP(new(bytes.Buffer), image.NewRGBA(image.Rect(0, 0, 1, 1)), 0)
	require.Error(t, err)
}
