package control

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexsergivan/transliterator"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

type addDirectoryRecursiveDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album]

	QueueBroker() *qlite.Broker
}

// AddDirectoryRecursive creates use case interactor to recursively add directory of photos to albums.
func AddDirectoryRecursive(deps addDirectoryRecursiveDeps, addDir usecase.IOInteractorOf[addDirInput, addDirOutput]) usecase.Interactor {
	type addRecursiveDirInput struct {
		Path string `formData:"path"`
	}

	type directory struct {
		Path      string `json:"path"`
		AlbumName string `json:"album_name"`
		Size      int64  `json:"size"`
		Count     int64  `json:"cnt"`
	}

	b := deps.QueueBroker()
	tr := transliterator.NewTransliterator(nil)
	replacer := strings.NewReplacer("/", "-", " ", "-", ".", "-", "--", "-")

	if err := qlite.AddConsumer[directory](b, "add-dir", func(ctx context.Context, v directory) error {
		dir := addDirInput{
			Path: v.Path,
			Name: v.AlbumName,
		}

		return addDir.Invoke(ctx, dir, &addDirOutput{})
	}); err != nil {
		panic(err)
	}

	if err := qlite.AddConsumer[directory](b, "add-dir-rec", func(ctx context.Context, v directory) error {
		files, err := os.ReadDir(v.Path)
		if err != nil {
			return err
		}

		isAlbum := false
		totalImages := int64(0)
		totalSize := int64(0)

		for _, f := range files {
			if f.IsDir() {
				err := b.Publish(ctx, "add-dir-rec", directory{
					Path: filepath.Join(v.Path, f.Name()),
				})
				if err != nil {
					return err
				}

				continue
			}

			lname := strings.ToLower(f.Name())
			if strings.HasSuffix(lname, ".jpg") || strings.HasSuffix(lname, ".jpeg") {
				isAlbum = true
				totalImages++
				fi, err := f.Info()
				if err != nil {
					return err
				}

				totalSize += fi.Size()
			}
		}

		albumName := strings.ToLower(tr.Transliterate(v.Path, "en"))
		albumName = replacer.Replace(albumName)
		albumName = strings.Trim(albumName, "-")

		if isAlbum {
			a := photo.Album{
				Name:  albumName,
				Title: v.Path,
			}
			a.Hash = uniq.StringHash(a.Name)

			if _, err := deps.PhotoAlbumEnsurer().Ensure(ctx, a); err != nil {
				return err
			}

			if err := b.Publish(ctx, "add-dir", directory{
				Path:      v.Path,
				AlbumName: albumName,
				Size:      totalSize,
				Count:     totalImages,
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in addRecursiveDirInput, out *addDirOutput) error {
		return b.Publish(ctx, "add-dir-rec", directory{Path: in.Path})
	})

	u.SetDescription("Recursively add a host-local directory of photos to albums.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
