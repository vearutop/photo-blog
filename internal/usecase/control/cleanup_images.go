package control

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

func CleanupRemote(deps interface {
	CtxdLogger() ctxd.Logger
	PhotoAlbumImageFinder() photo.AlbumImageFinder
},
) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *struct{}) error {
		images, err := deps.PhotoAlbumImageFinder().FindRemoteImages(ctx)
		if err != nil {
			return err
		}

		deps.CtxdLogger().Info(ctx, "cleanup remote images", "found", len(images))
		moved := 0
		noAlbumPrefix := 0
		noHTTPSources := 0
		noFile := 0
		failed := 0

		dirsCreated := make(map[string]bool)

		for _, i := range images {
			if len(i.Settings.HTTPSources) == 0 {
				noHTTPSources++

				continue
			}

			if !strings.HasPrefix(i.Path, "album/") {
				noAlbumPrefix++

				continue
			}

			_, err := os.Lstat(i.Path)
			if os.IsNotExist(err) {
				noFile++

				continue
			}

			if err != nil {
				failed++

				deps.CtxdLogger().Error(ctx, "failed to stat image path", "img", i)
				continue
			}

			newPath := "check/" + strings.TrimPrefix(i.Path, "album/")

			dir := filepath.Dir(newPath)
			if !dirsCreated[dir] {
				err := os.MkdirAll(dir, 0o700)
				if err != nil {
					failed++

					return err
				}

				dirsCreated[dir] = true
			}

			if err := os.Rename(i.Path, newPath); err != nil {
				failed++
				deps.CtxdLogger().Error(ctx, "failed to cleanup remote image", "img", i, "err", err.Error())
			} else {
				moved++
				deps.CtxdLogger().Info(ctx, "remote image moved to check", "img", i)
			}

		}

		deps.CtxdLogger().Info(ctx, "cleanup remote images",
			"moved", moved,
			"failed", failed,
			"noAlbumPrefix", noAlbumPrefix,
			"noHTTPSources", noHTTPSources,
			"noFile", noFile,
		)

		return nil
	})

	u.SetDescription("Cleanup iterates remote images moves existing local copies to 'check' directory.")

	return u
}
