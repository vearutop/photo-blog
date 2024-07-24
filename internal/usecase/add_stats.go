package usecase

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/webstats"
)

type collectStatsDeps interface {
	CtxdLogger() ctxd.Logger
	VisitorStats() *visitor.StatsRepository
}

func CollectStats(deps collectStatsDeps) usecase.Interactor {
	// /stats?album=featured&img=3d3kr8ydrb8l4&time=4169&w=1475&h=983&mw=1620&mh=1080&sw=1920&sh=1080&px=2&v=k...g
	// /stats?album=featured&img=1om86jo4rku2r&time=391&w=1620&h=1080&mw=1621&mh=1080&sw=1920&sh=1080&px=2&v=k...g
	// /stats?main=1&sw=1920&sh=1080&px=2&v=k...g
	// /stats?thumb=%7B%2234suxvlfx0lz8%22%3A36704%2C%221z4zoegvmke8n%22%3A36704%2C%223b45tgt52cnms%22%3A36704%2C%221d2ujpqi6nbb4%22%3A36704%2C%221shlwpftv8av4%22%3A36704%7D&sw=1792&sh=1120&px=2&v=1...w
	type collectStatsRequest struct {
		request.EmbeddedSetter

		Visitor uniq.Hash `query:"v" description:"Visitor."`
		Referer string    `query:"ref" description:"Referer."`

		ScreenWidth  int     `query:"sw" description:"Screen width."`
		ScreenHeight int     `query:"sh" description:"Screen height."`
		PixelRatio   float64 `query:"px" description:"Device pixel ratio (retina factor)."`

		Main               bool              `query:"main" description:"Main page shown."`
		Album              string            `query:"album" description:"Album with a name shown."`
		Thumb              map[uniq.Hash]int `query:"thumb" collectionFormat:"json" description:"Thumb on-screen times, ms."`
		MobilePortraitMode bool              `query:"prt" description:"Mobile portrait mode."`

		Image     uniq.Hash `query:"img" description:"Image with a hash is shown individually."`
		Width     int       `query:"w" description:"Shown width of the image."`
		Height    int       `query:"h" description:"Shown height of the image."`
		MaxWidth  int       `query:"mw" description:"Max shown width of the image (zoom)."`
		MaxHeight int       `query:"mh" description:"Max shown height of the image (zoom)."`
		Time      int       `query:"time" description:"Image view time, ms."`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input collectStatsRequest, output *struct{}) error {
		deps.CtxdLogger().Info(ctx, "stats", "input", input, "admin", auth.IsAdmin(ctx))

		if webstats.IsBot(input.Request().UserAgent()) || deps.VisitorStats().IsAdmin(input.Visitor) {
			return nil
		}

		switch {
		// /stats?main=1&sw=1920&sh=1080&px=2&v=k...g
		case input.Main:
			deps.VisitorStats().CollectMain(ctx, input.Visitor, input.Referer, time.Now())

		// /stats?album=featured&img=3d3kr8ydrb8l4&time=4169&w=1475&h=983&mw=1620&mh=1080&sw=1920&sh=1080&px=2&v=k...g
		case input.Image != 0:
			zoomedIn := float64(input.MaxWidth)/float64(input.Width) > 1.1 // At least 10% zoom in.
			deps.VisitorStats().CollectImage(ctx, input.Visitor, input.Image, input.Time, zoomedIn)

		// /stats?album=2024-07-13-aloevera&sw=1280&sh=800&px=2&v=qtuf2cgx08i4
		case input.Album != "":
			deps.VisitorStats().CollectAlbum(ctx, input.Visitor, photo.AlbumHash(input.Album), input.Referer, time.Now())

		// /stats?thumb=%7B%2234suxvlfx0lz8%22%3A36704%2C%221z4zoegvmke8n%22%3A36704%2C%223b45tgt52cnms%22%3A36704%2C%221d2ujpqi6nbb4%22%3A36704%2C%221shlwpftv8av4%22%3A36704%7D&sw=1792&sh=1120&px=2&v=1...w
		case len(input.Thumb) > 0:
			deps.VisitorStats().CollectThumbs(ctx, input.Visitor, input.MobilePortraitMode, input.Thumb)
		}

		if input.Referer != "" {
			deps.VisitorStats().CollectRefer(ctx, input.Visitor, time.Now(), input.Referer, input.Request().URL.String())
		}

		return nil
	})

	return u
}
