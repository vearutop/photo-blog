package upload

import (
	"context"
	"crypto/tls"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/web"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

func MountTus(s *web.Service, deps TusHandlerDeps) error {
	store := filestore.FileStore{
		Path: deps.ServiceSettings().UploadStorage + "/temp",
	}
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	tusHandler, err := tusd.NewHandler(tusd.Config{
		BasePath:                "/files",
		RespectForwardedHeaders: true,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			event := <-tusHandler.CompleteUploads
			processUpload(deps, event)
		}
	}()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps.CtxdLogger().Info(r.Context(), "request headers", r.Header)

		if strings.HasPrefix(r.Header.Get("Origin"), "https://") {
			r.TLS = &tls.ConnectionState{}
		}

		tusHandler.ServeHTTP(w, r)
	})

	s.Mount("/files/", http.StripPrefix("/files/", h))
	s.Method(http.MethodPost, "/files", http.StripPrefix("/files", h))

	return nil
}

func processUpload(deps TusHandlerDeps, event tusd.HookEvent) {
	ctx := context.Background()
	deps.CtxdLogger().Info(ctx, "upload finished", "event", event)

	defer func() {
		if err := os.Remove(deps.ServiceSettings().UploadStorage + "/temp/" + event.Upload.ID + ".info"); err != nil {
			deps.CtxdLogger().Error(ctx, "failed to remove uploaded info", "error", err)
		}
		if err := os.Remove(deps.ServiceSettings().UploadStorage + "/temp/" + event.Upload.ID); err != nil {
			deps.CtxdLogger().Error(ctx, "failed to remove uploaded file", "error", err)
		}
	}()

	albumName := event.HTTPRequest.Header.Get("X-Album-Name")
	if albumName == "" {
		deps.CtxdLogger().Error(ctx, "no album name in upload", "event", event)

		return
	}

	albumPath := path.Join(deps.ServiceSettings().UploadStorage, albumName)
	if err := os.MkdirAll(albumPath, 0o700); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to create album directory", "error", err)

		return
	}

	filePath := path.Join(albumPath, event.Upload.MetaData["filename"])
	if err := os.Rename(event.Upload.Storage["Path"], filePath); err != nil {
		deps.CtxdLogger().Error(ctx, "failed to create album directory", "error", err)

		return
	}

	if err := deps.FilesProcessor().AddFile(ctx, albumName, filePath); err != nil {
		if errors.Is(err, files.ErrSkip) {
			if err := os.Remove(filePath); err != nil {
				deps.CtxdLogger().Error(ctx, "failed to remove skipped file",
					"error", err,
					"filePath", filePath)
			}
		}
	}
}

type TusHandlerDeps interface {
	CtxdLogger() ctxd.Logger
	ServiceSettings() service.Settings
	FilesProcessor() *files.Processor
}

func TusAlbumHTMLButton(albumName string) template.HTML {
	return template.HTML(`
<button style="margin: 2em" class="btn btn-secondary" id="uppyModalOpener">Upload files</button>
<script>
    {
        const { Dashboard, Tus } = Uppy
        const uppy = new Uppy.Uppy({ debug: true, autoProceed: false })
            .use(Dashboard, { 
				trigger: '#uppyModalOpener', 
				note: 'JPG, GPX are supported', 
				proudlyDisplayPoweredByUppy: false,
			})
            .use(Tus, { 
				endpoint: window.location.protocol + '//' + window.location.host + '/files',
				chunkSize: 900000, // 900K to fit in 1MiB default client_max_body_size of nginx.
				headers: {"X-Album-Name": "` + albumName + `"},
			})
    }
</script>
`)
}
