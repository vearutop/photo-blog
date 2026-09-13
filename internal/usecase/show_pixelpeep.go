package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/pixelpeep"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type pixelpeepDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
}

type showPixelpeepInput struct {
	Hashes string `path:"hashes"`
}

// ShowPixelpeep renders the pixelpeep editor for 2-4 images given by
// comma-separated hash, e.g. /pixelpeep-<hash1>,<hash2>. Use its "Copy link"
// button to share a read-only view (see SharePixelpeep).
func ShowPixelpeep(deps pixelpeepDeps) usecase.Interactor {
	tmpl, err := static.Template("pixelpeep.html")
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showPixelpeepInput, out *web.Page) error {

		cfg := pixelpeep.Config{}
		for s := range strings.SplitSeq(in.Hashes, ",") {
			var h uniq.Hash

			if err := h.UnmarshalText([]byte(s)); err != nil {
				return status.Wrap(fmt.Errorf("decode hash %q: %w", s, err), status.InvalidArgument)
			}

			cfg.Items = append(cfg.Items, pixelpeep.Image{URL: "/image/" + h.String() + ".jpg"})
		}

		if err := cfg.Validate(); err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}

		return renderPixelpeep(out, tmpl, cfg, true, "/pixelpeep")
	})

	u.SetTags("Pixelpeep")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

type sharePixelpeepInput struct {
	Config string `query:"config"`
}

// SharePixelpeep renders a read-only pixelpeep view from a JSON config passed
// as a query parameter, e.g. /pixelpeep?config=... (as produced by "Copy
// link" in the editor from ShowPixelpeep).
func SharePixelpeep() usecase.Interactor {
	tmpl, err := static.Template("pixelpeep.html")
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in sharePixelpeepInput, out *web.Page) error {
		var cfg pixelpeep.Config

		if err := json.Unmarshal([]byte(in.Config), &cfg); err != nil {
			return status.Wrap(fmt.Errorf("decode config: %w", err), status.InvalidArgument)
		}

		if err := cfg.Validate(); err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}

		return renderPixelpeep(out, tmpl, cfg, false, "")
	})

	u.SetTags("Pixelpeep")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

func renderPixelpeep(out *web.Page, tmpl *template.Template, cfg pixelpeep.Config, editor bool, shareBase string) error {
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	data := struct {
		ConfigJSON template.JS
		Editor     bool
		ShareBase  string
	}{ConfigJSON: template.JS(configJSON), Editor: editor, ShareBase: shareBase}

	return out.Render(tmpl, data)
}
