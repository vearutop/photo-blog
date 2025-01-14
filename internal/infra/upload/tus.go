package upload

import (
	"context"
	"crypto/tls"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/web"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"golang.org/x/exp/slog"
)

func MountTus(s *web.Service, deps TusHandlerDeps) error {
	store := filestore.FileStore{
		Path: "temp",
	}
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	tusHandler, err := tusd.NewHandler(tusd.Config{
		BasePath:                "/files",
		RespectForwardedHeaders: true,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		Logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)), // TODO: use ctxd.Logger.
	})
	if err != nil {
		return err
	}

	up := &uploadProcessor{
		thumbWait: make(map[string]thumbWaiter),
	}

	go func() {
		for {
			event := <-tusHandler.CompleteUploads
			up.processUpload(deps, event)
		}
	}()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// deps.CtxdLogger().Info(r.Context(), "request headers", "header", r.Header)

		if strings.HasPrefix(r.Header.Get("Origin"), "https://") {
			r.TLS = &tls.ConnectionState{}
		}

		if !auth.IsAdmin(r.Context()) {
			albumName := r.Header.Get("X-Album-Name")
			collabKey := r.Header.Get("X-Collab-Key")

			if collabKey == "" || albumName == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			album, err := deps.PhotoAlbumFinder().FindByHash(r.Context(), photo.AlbumHash(albumName))
			if err != nil || album.Settings.CollabKey != collabKey {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		tusHandler.ServeHTTP(w, r)
	})

	s.Mount("/files/", http.StripPrefix("/files/", h))
	s.Method(http.MethodPost, "/files", http.StripPrefix("/files", h))

	return nil
}

type uploadProcessor struct {
	mu        sync.Mutex
	thumbWait map[string]thumbWaiter // Key is album-name:file-name
}

type thumbWaiter struct {
	thumbsLeft []string
	hash       uniq.Hash
	idx        func()
}

func (up *uploadProcessor) processUpload(deps TusHandlerDeps, event tusd.HookEvent) {
	ctx := event.Context
	deps.CtxdLogger().Info(ctx, "upload finished", "event", event)

	defer func() {
		if err := os.Remove("temp/" + event.Upload.ID + ".info"); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				deps.CtxdLogger().Error(ctx, "failed to remove uploaded info", "error", err)
			}
		}
		if err := os.Remove("temp/" + event.Upload.ID); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				deps.CtxdLogger().Error(ctx, "failed to remove uploaded file", "error", err)
			}
		}
	}()

	albumName := event.HTTPRequest.Header.Get("X-Album-Name")
	if albumName == "" { // Upload to /site.
		siteUpload(ctx, deps, event)

		return
	}

	albumPath := AlbumPath(albumName)
	if err := os.MkdirAll(albumPath, 0o700); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to create album directory", "error", err)

		return
	}

	md := event.Upload.MetaData

	filePath := AlbumFilePath(albumPath, md["filename"])

	// Check for thumbnail.
	if relPath := md["relativePath"]; relPath != "null" {
		for _, th := range photo.ThumbSizes {
			if strings.Contains(relPath, string(th)) {
				deps.CtxdLogger().Info(ctx, "thumbnail uploaded", "album", albumName, "path", filePath, "size", th)

				up.mu.Lock()
				defer up.mu.Unlock()

				tw := up.thumbWait[filePath]

				tl := tw.thumbsLeft
				for i, thl := range tl {
					if string(th) == thl {
						tl[i] = tl[len(tl)-1]
						tl = tl[:len(tl)-1]

						break
					}
				}
				tw.thumbsLeft = tl

				if err := deps.FilesProcessor().AddThumbnail(ctx, tw.hash, th, event.Upload.Storage["Path"]); err != nil {
					deps.CtxdLogger().Error(ctx, "failed to add uploaded thumb", "error", err)
					return
				}

				if len(tl) == 0 {
					deps.CtxdLogger().Info(ctx, "all thumbs uploaded", "album", albumName, "path", filePath, "size", th)
					if tw.idx != nil {
						tw.idx()
					}

					delete(up.thumbWait, filePath)
				} else {
					up.thumbWait[filePath] = tw
				}

				return
			}
		}
	}

	if err := os.Rename(event.Upload.Storage["Path"], filePath); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to move uploaded file", "error", err)

		return
	}

	tw := thumbWaiter{
		thumbsLeft: strings.Split(event.HTTPRequest.Header.Get("X-Expect-Thumbnails"), ","),
	}

	var err error

	if tw.hash, tw.idx, err = deps.FilesProcessor().AddFile(ctx, albumName, filePath); err != nil {
		if errors.Is(err, files.ErrSkip) {
			if err := os.Remove(filePath); err != nil {
				deps.CtxdLogger().Error(ctx, "failed to remove skipped file",
					"error", err,
					"filePath", filePath)
			}
		} else {
			deps.CtxdLogger().Error(ctx, "failed to process uploaded file",
				"error", err,
				"filePath", filePath)
		}
	} else {
		if len(tw.thumbsLeft) > 0 {
			up.mu.Lock()
			defer up.mu.Unlock()

			up.thumbWait[filePath] = tw
		} else {
			tw.idx()
		}
	}
}

func AlbumPath(albumName string) string {
	return path.Join("album", albumName)
}

func AlbumFilePath(albumPath, fileName string) string {
	return albumPath + "/" + fileName
}

func siteUpload(ctx context.Context, deps TusHandlerDeps, event tusd.HookEvent) {
	dir := "site"

	if err := os.MkdirAll(dir, 0o700); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to create site directory", "error", err)

		return
	}

	filePath := path.Join(dir, event.Upload.MetaData["filename"])
	if err := os.Rename(event.Upload.Storage["Path"], filePath); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to move uploaded file", "error", err)

		return
	}
}

type TusHandlerDeps interface {
	CtxdLogger() ctxd.Logger
	FilesProcessor() *files.Processor
	PhotoAlbumFinder() uniq.Finder[photo.Album]
}

func TusUploadsButton() template.HTML {
	return `
<button style="margin: 2em" class="btn btn-secondary" id="uppyModalOpener">Upload site files</button>
<script>
    {
        const { Dashboard, Tus } = Uppy
        const uppy = new Uppy.Uppy({ debug: true, autoProceed: false })
            .use(Dashboard, { 
				trigger: '#uppyModalOpener', 
				note: 'These files would be available with "/site/<name.ext>" HTTP(s) links.', 
				proudlyDisplayPoweredByUppy: false,
			})
            .use(Tus, { 
				endpoint: window.location.protocol + '//' + window.location.host + '/files',
				chunkSize: 900000, // 900K to fit in 1MiB default client_max_body_size of nginx.
			})
    }
</script>
`
}

func TusAlbumHTMLButton(albumName string) template.HTML {
	return template.HTML(`
<button style="margin: 2em" class="btn btn-secondary" id="uppyModalOpener">Upload files</button>
<script>
    {
        const { Dashboard, Tus } = Uppy
        const uppy = new Uppy.Uppy({ debug: true, autoProceed: false, limit: 1 })
            .use(Dashboard, { 
				trigger: '#uppyModalOpener', 
				note: 'JPG, GPX are supported', 
				proudlyDisplayPoweredByUppy: false,
			})
            .use(Tus, { 
				limit: 1,
				endpoint: window.location.protocol + '//' + window.location.host + '/files',
				chunkSize: 900000, // 900K to fit in 1MiB default client_max_body_size of nginx.
				headers: {"X-Album-Name": "` + albumName + `"},
				onBeforeRequest: function (req, file) {
					beforeUploadRequest(req, file, uppy.getFiles())
				}
			})
    }
</script>
`)
}
