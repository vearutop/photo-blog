package usecase

import (
	"archive/zip"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"net/http"
	"os"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

const maxSplitGrid = 20

type splitImageDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	Settings() settings.Values
}

type splitImageInput struct {
	Grid int       `path:"grid"`
	Hash uniq.Hash `path:"hash"`
	request.EmbeddedSetter
}

// SplitImage downloads an image as a ZIP archive containing Grid by Grid JPEG tiles.
func SplitImage(deps splitImageDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in splitImageInput, out *response.EmbeddedSetter) error {
		if in.Grid < 1 || in.Grid > maxSplitGrid {
			return status.Wrap(fmt.Errorf("grid must be between 1 and %d", maxSplitGrid), status.InvalidArgument)
		}

		if deps.Settings().Privacy().HideOriginal && !auth.IsAdmin(ctx) {
			exif, err := deps.PhotoExifFinder().FindByHash(ctx, in.Hash)
			if err != nil {
				return err
			}
			if exif.ProjectionType == "" {
				return status.PermissionDenied
			}
		}

		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		src, err := openSplitImage(ctx, img)
		if err != nil {
			return err
		}
		defer src.Close()

		decoded, _, err := image.Decode(src)
		if err != nil {
			return fmt.Errorf("decode image: %w", err)
		}
		if decoded.Bounds().Dx() < in.Grid || decoded.Bounds().Dy() < in.Grid {
			return status.Wrap(fmt.Errorf("grid %d exceeds image dimensions %dx%d", in.Grid, decoded.Bounds().Dx(), decoded.Bounds().Dy()), status.InvalidArgument)
		}

		rw := out.ResponseWriter()
		rw.Header().Set("Content-Type", "application/zip")
		rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fmt.Sprintf("%s-%dx%d.zip", in.Hash, in.Grid, in.Grid)))
		rw.Header().Set("Cache-Control", "max-age=31536000")

		if err := writeImageSplitZIP(rw, decoded, in.Grid); err != nil {
			return fmt.Errorf("write image split ZIP: %w", err)
		}

		return nil
	})
	u.SetTags("Image")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument, status.PermissionDenied)

	return u
}

func openSplitImage(ctx context.Context, img photo.Image) (io.ReadCloser, error) {
	if len(img.Settings.HTTPSources) == 0 {
		return os.Open(img.Path)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, img.Settings.HTTPSources[0], nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch remote image: %s", resp.Status)
	}

	return resp.Body, nil
}

func writeImageSplitZIP(dst io.Writer, src image.Image, grid int) error {
	if grid < 1 {
		return fmt.Errorf("grid must be positive")
	}

	zw := zip.NewWriter(dst)
	bounds := src.Bounds()

	for row := range grid {
		y0 := bounds.Min.Y + bounds.Dy()*row/grid
		y1 := bounds.Min.Y + bounds.Dy()*(row+1)/grid

		for col := range grid {
			x0 := bounds.Min.X + bounds.Dx()*col/grid
			x1 := bounds.Min.X + bounds.Dx()*(col+1)/grid

			tile := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
			draw.Draw(tile, tile.Bounds(), src, image.Point{X: x0, Y: y0}, draw.Src)

			entry, err := zw.Create(fmt.Sprintf("row-%02d-col-%02d.jpg", row+1, col+1))
			if err != nil {
				return err
			}
			if err := jpeg.Encode(entry, tile, &jpeg.Options{Quality: 95}); err != nil {
				return err
			}
		}
	}

	return zw.Close()
}
