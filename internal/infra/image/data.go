package image

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"os"
	"time"

	"github.com/bool64/ctxd"
	"github.com/buckket/go-blurhash"
	"github.com/corona10/goimagehash"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
)

type Data struct {
	Image  photo.Image   `json:"image"`
	Exif   *photo.Exif   `json:"exif,omitempty"`
	Gps    *photo.Gps    `json:"gps,omitempty"`
	Meta   *photo.Meta   `json:"meta,omitempty"`
	Thumbs []photo.Thumb `json:"thumbs,omitempty"`
}

func (d *Data) Fill(ctx context.Context) error {
	img := &d.Image

	if img.Hash == 0 {
		if err := img.SetPath(ctx, img.Path); err != nil {
			return fmt.Errorf("set path: %w", err)
		}
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now()
	}

	if err := d.resolution(ctx); err != nil {
		return fmt.Errorf("resolution: %w", err)
	}

	if err := d.exif(ctx); err != nil {
		return fmt.Errorf("exif: %w", err)
	}

	d.takenAt()

	if err := d.thumbs(ctx); err != nil {
		return fmt.Errorf("thumbs: %w", err)
	}

	if err := d.imgHash(ctx); err != nil {
		return fmt.Errorf("imgHash: %w", err)
	}

	return nil
}

func (d *Data) takenAt() {
	img := &d.Image

	if img.TakenAt == nil {
		switch {
		case d.Exif != nil && d.Exif.Digitized != nil:
			img.TakenAt = d.Exif.Digitized
		case d.Gps != nil && !d.Gps.GpsTime.IsZero():
			img.TakenAt = &d.Gps.GpsTime
		default:
			img.TakenAt = &img.CreatedAt
		}
	}
}

func (d *Data) imgHash(ctx context.Context) error {
	if d.Image.BlurHash != "" && d.Image.PHash != 0 {
		return nil
	}

	for _, th := range d.Thumbs {
		if th.Format != "300w" {
			continue
		}

		j, err := thumbJPEG(ctx, th)
		if err != nil {
			return err
		}

		bh, err := blurhash.Encode(5, 5, j)
		if err != nil {
			return err
		}

		d.Image.BlurHash = bh

		h, err := goimagehash.PerceptionHash(j)
		if err != nil {
			return err
		}

		d.Image.PHash = uniq.Hash(h.GetHash())

		return nil
	}

	return errors.New("300w thumb not found")
}

func (d *Data) thumbs(ctx context.Context) error {
	for _, size := range photo.ThumbSizes {
		th, err := makeThumbnail(ctx, d.Image, size)
		if err != nil {
			return err
		}

		d.Thumbs = append(d.Thumbs, th)
		ctx = LargerThumbToContext(ctx, th)
	}

	return nil
}

func (d *Data) exif(ctx context.Context) (err error) {
	if d.Exif != nil {
		return nil
	}

	img := &d.Image

	f, err := os.Open(img.Path)
	if err != nil {
		return ctxd.WrapError(ctx, err, "open image file")
	}

	defer func() {
		if clErr := f.Close(); clErr != nil {
			err = errors.Join(err, clErr)
		}
	}()

	m, err := ReadMeta(f)
	if err != nil {
		return ctxd.WrapError(ctx, err, "read image meta")
	}

	exifQuirks(&m.Exif)

	m.Exif.Hash = img.Hash
	d.Exif = &m.Exif
	d.Gps = m.GpsInfo

	return nil
}

func (d *Data) resolution(ctx context.Context) (err error) {
	img := &d.Image

	if img.Width > 0 {
		return nil
	}

	f, err := os.Open(img.Path)
	if err != nil {
		return ctxd.WrapError(ctx, err, "open image file")
	}
	defer func() {
		if clErr := f.Close(); clErr != nil {
			err = errors.Join(err, clErr)
		}
	}()

	c, err := jpeg.DecodeConfig(f)
	if err != nil {
		return ctxd.WrapError(ctx, err, "image dimensions")
	}

	img.Width = int64(c.Width)
	img.Height = int64(c.Height)

	return nil
}

func (d *Data) FillFaces(ctx context.Context, rec *faces.Recognizer) error {
	return nil
}
