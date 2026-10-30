package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showRoomPreviewInput struct {
	Hashes string `path:"hashes"`
}

// ShowRoomPreview renders the room-preview editor for a photo of an empty room and a sample photo given by
// comma-separated hash, e.g. /room-preview-<room hash>,<sample hash>. Its output is a widget config for album
// markdown, see room-preview.js.
func ShowRoomPreview() usecase.Interactor {
	tmpl, err := static.Template("room-preview.html")
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showRoomPreviewInput, out *web.Page) error {
		hashes := strings.Split(in.Hashes, ",")
		if len(hashes) != 2 {
			return status.Wrap(fmt.Errorf("need room and sample image hashes, got %d", len(hashes)), status.InvalidArgument)
		}

		for _, s := range hashes {
			var h uniq.Hash

			if err := h.UnmarshalText([]byte(s)); err != nil {
				return status.Wrap(fmt.Errorf("decode hash %q: %w", s, err), status.InvalidArgument)
			}
		}

		cfg, err := json.Marshal(map[string]any{"room": hashes[0], "items": []string{hashes[1]}})
		if err != nil {
			return err
		}

		return out.Render(tmpl, struct{ ConfigJSON template.JS }{ConfigJSON: template.JS(cfg)})
	})

	u.SetTags("RoomPreview")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
