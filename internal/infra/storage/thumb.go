package storage

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	image "github.com/vearutop/photo-blog/internal/infra/image"
)

const (
	// ThumbTable is the name of the table.
	ThumbTable = "thumb"
)

func NewThumbRepository(storage *sqluct.Storage, upstream photo.Thumbnailer, logger ctxd.Logger) *ThumbRepository {
	return &ThumbRepository{
		upstream: upstream,
		logger:   logger,
		hashedRepo: hashedRepo[photo.Thumb, *photo.Thumb]{
			StorageOf: sqluct.Table[photo.Thumb](storage, ThumbTable),
		},
	}
}

// ThumbRepository saves images to database.
type ThumbRepository struct {
	upstream photo.Thumbnailer
	logger   ctxd.Logger
	hashedRepo[photo.Thumb, *photo.Thumb]
}

func (tr *ThumbRepository) Thumbnail(ctx context.Context, img photo.Image, size photo.ThumbSize) (photo.Thumb, error) {
	th := photo.Thumb{}

	w, h, err := size.WidthHeight()
	if err != nil {
		return th, err
	}

	th, err = tr.Find(ctx, img.Hash, w, h)
	if err == nil {
		return th, nil
	}

	if lt := image.LargerThumbFromContext(ctx); lt == nil || lt.Format != size {
		if lt, err := tr.FindLarger(ctx, img.Hash, w, h); err == nil {
			ctx = image.LargerThumbToContext(ctx, lt)
		}
	}

	th, err = tr.upstream.Thumbnail(ctx, img, size)
	if err != nil {
		return th, err
	}

	if err := tr.Add(ctx, th); err != nil {
		return th, augmentErr(err)
	}

	return th, nil
}

func (tr *ThumbRepository) FindLarger(ctx context.Context, imageHash uniq.Hash, width, height uint) (photo.Thumb, error) {
	q := tr.SelectStmt().
		Where(tr.Eq(&tr.R.Hash, imageHash))

	if width > 0 {
		q = q.Where(squirrel.GtOrEq(tr.Eq(&tr.R.Width, width)))
	}

	if height > 0 {
		q = q.Where(squirrel.GtOrEq(tr.Eq(&tr.R.Height, height)))
	}

	row, err := tr.Get(ctx, q)
	if err != nil {
		return photo.Thumb{}, fmt.Errorf("find thumb by image %q and size %dx%d: %w",
			imageHash, width, height, augmentErr(err))
	}

	return row, nil
}

func (tr *ThumbRepository) Find(ctx context.Context, imageHash uniq.Hash, width, height uint) (photo.Thumb, error) {
	q := tr.SelectStmt().
		Where(tr.Eq(&tr.R.Hash, imageHash))

	if width > 0 {
		q = q.Where(tr.Eq(&tr.R.Width, width))
	}

	if height > 0 {
		q = q.Where(tr.Eq(&tr.R.Height, height))
	}

	row, err := tr.Get(ctx, q)
	if err != nil {
		return photo.Thumb{}, fmt.Errorf("find thumb by image %q and size %dx%d: %w",
			imageHash, width, height, augmentErr(err))
	}

	return row, nil
}

func (tr *ThumbRepository) PhotoThumbnailer() photo.Thumbnailer {
	return tr
}
