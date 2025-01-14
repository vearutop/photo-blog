package files

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

type ProcessorDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
	PhotoImageIndexer() photo.ImageIndexer

	PhotoGpxEnsurer() uniq.Ensurer[photo.Gpx]

	PhotoThumbnailer() photo.Thumbnailer

	DepCache() *dep.Cache
}

func NewProcessor(deps ProcessorDeps) *Processor {
	return &Processor{deps: deps}
}

type Processor struct {
	deps ProcessorDeps
}

const (
	ErrSkip = ctxd.SentinelError("unsupported file skipped")
)

func (p *Processor) AddThumbnail(ctx context.Context, imgHash uniq.Hash, size photo.ThumbSize, filePath string) error {
	img := photo.Image{}
	img.Hash = imgHash

	var err error

	th := photo.Thumb{}
	th.Hash = imgHash
	th.Format = size
	th.Data, err = os.ReadFile(filePath)
	if err != nil {
		return err
	}

	ctx = image.LargerThumbToContext(ctx, th)

	p.deps.PhotoThumbnailer().Thumbnail(ctx, img, size)

	return nil
}

func (p *Processor) AddFile(ctx context.Context, albumName string, filePath string, after ...func(hash uniq.Hash)) (h uniq.Hash, idx func(), err error) {
	lName := strings.ToLower(filePath)

	defer func() {
		if err == nil {
			err = p.deps.DepCache().AlbumChanged(ctx, albumName)
		}
	}()

	if strings.HasSuffix(lName, ".jpg") || strings.HasSuffix(lName, ".jpeg") {
		d := photo.Image{}
		if err := d.SetPath(ctx, filePath); err != nil {
			return 0, nil, fmt.Errorf("set image path: %w", err)
		}

		img, err := p.deps.PhotoImageEnsurer().Ensure(ctx, d)
		if err != nil {
			return 0, nil, fmt.Errorf("ensure image: %w", err)
		} else if err := p.deps.PhotoAlbumImageAdder().AddImages(ctx, uniq.StringHash(albumName), img.Hash); err != nil {
			return 0, nil, fmt.Errorf("add image to album: %w", err)
		}
		return img.Hash, func() {
			p.deps.PhotoImageIndexer().QueueIndex(ctx, img, photo.IndexingFlags{})
			p.deps.PhotoImageIndexer().QueueCallback(ctx, func(ctx context.Context) {
				for _, cb := range after {
					cb(img.Hash)
				}
				_ = p.deps.DepCache().AlbumChanged(ctx, albumName)
			})
		}, nil
	}

	if strings.HasSuffix(lName, ".gpx") {
		d := photo.Gpx{}
		if err := d.SetPath(ctx, filePath); err != nil {
			return 0, nil, fmt.Errorf("set gpx path: %w", err)
		}

		if err := d.Index(); err != nil {
			return 0, nil, fmt.Errorf("index gpx: %w", err)
		}

		p.deps.CtxdLogger().Info(ctx, "gpx", "settings", d.Settings.Val)

		d, err := p.deps.PhotoGpxEnsurer().Ensure(ctx, d)
		if err != nil {
			return 0, nil, fmt.Errorf("ensure gpx: %w", err)
		} else {
			// TODO: migrate album_images to album_contents with hashed items of different types
			// (e.g. gpx, or gps poi, or even comments/descriptions?).
			a, err := p.deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(albumName))
			if err != nil {
				return 0, nil, fmt.Errorf("find album %s: %w", albumName, err)
			}

			found := false

			for _, hh := range a.Settings.GpxTracksHashes {
				if d.Hash == hh {
					found = true
					break
				}
			}

			if !found {
				a.Settings.GpxTracksHashes = append(a.Settings.GpxTracksHashes, d.Hash)

				if _, err = p.deps.PhotoAlbumEnsurer().Ensure(ctx, a); err != nil {
					return 0, nil, fmt.Errorf("ensure album %s: %w", albumName, err)
				}
			}

			for _, cb := range after {
				cb(d.Hash)
			}

			return d.Hash, nil, nil
		}
	}

	return 0, nil, ErrSkip
}

func (p *Processor) AddDirectory(ctx context.Context, albumName string, dirPath string) ([]string, error) {
	p.deps.StatsTracker().Add(ctx, "add_dir", 1)
	p.deps.CtxdLogger().Important(ctx, "adding directory", "path", dirPath)

	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, nil
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "read dir names", "path", dirPath)
	}

	p.deps.CtxdLogger().Info(ctx, "directory contents", "names", names)

	var (
		added []string
		errs  []string
	)

	for _, name := range names {
		if _, idx, err := p.AddFile(ctx, albumName, path.Join(dirPath, name)); err != nil {
			if !errors.Is(err, ErrSkip) {
				errs = append(errs, name+": "+err.Error())
			}
		} else {
			added = append(added, name)
			if idx != nil {
				idx()
			}
		}
	}

	if len(errs) > 0 {
		return added, ctxd.NewError(ctx, "there were errors", "errors", errs)
	}

	return added, err
}
