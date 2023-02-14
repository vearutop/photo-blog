package storage

import (
	"context"
	"fmt"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// ThumbTable is the name of the table.
	ThumbTable = "thumb"
)

func NewThumbRepository(storage *sqluct.Storage, upstream photo.Thumbnailer) *ThumbRepository {
	return &ThumbRepository{
		upstream: upstream,
		hashedRepo: hashedRepo[photo.Thumb, *photo.Thumb]{
			StorageOf: sqluct.Table[photo.Thumb](storage, ThumbTable),
		},
	}
}

// ThumbRepository saves images to database.
type ThumbRepository struct {
	upstream photo.Thumbnailer
	hashedRepo[photo.Thumb, *photo.Thumb]
}

func (tr *ThumbRepository) Thumbnail(ctx context.Context, image photo.Image, size photo.ThumbSize) (photo.Thumb, error) {
	th := photo.Thumb{}

	w, h, err := size.WidthHeight()
	if err != nil {
		return th, err
	}

	th, err = tr.Find(ctx, image.Hash, w, h)
	if err == nil {
		return th, nil
	}

	th, err = tr.upstream.Thumbnail(ctx, image, size)
	if err != nil {
		return th, err
	}

	if err := tr.Add(ctx, th); err != nil {
		return th, augmentErr(err)
	}

	return th, nil
}

func (tr *ThumbRepository) Find(ctx context.Context, imageHash uniq.Hash, width, height uint) (photo.Thumb, error) {
	q := tr.SelectStmt().
		Where(tr.Eq(&tr.R.Hash, imageHash))

	if width > 0 {
		q = q.Where(tr.Fmt("%s = %d", &tr.R.Width, width))
	}

	if height > 0 {
		q = q.Where(tr.Fmt("%s = %d", &tr.R.Height, height))
	}

	row, err := tr.Get(ctx, q)
	if err != nil {
		return photo.Thumb{}, fmt.Errorf("find thumb by image %q and size %dx%d: %w",
			imageHash, width, height, augmentErr(err))
	}

	return row, nil
}

func (tr *ThumbRepository) PhotoThumbEnsurer() uniq.Ensurer[photo.Thumb] {
	return tr
}

func (tr *ThumbRepository) PhotoThumbFinder() uniq.Finder[photo.Thumb] {
	return tr
}

func (tr *ThumbRepository) PhotoThumbAdder() uniq.Adder[photo.Thumb] {
	return tr
}

func (tr *ThumbRepository) PhotoThumbnailer() photo.Thumbnailer {
	return tr
}
