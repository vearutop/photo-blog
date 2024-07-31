package visitor

import (
	"github.com/swaggest/rest/request"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type CollectStats struct {
	// /stats?album=featured&img=3d3kr8ydrb8l4&time=4169&w=1475&h=983&mw=1620&mh=1080&sw=1920&sh=1080&px=2&v=k...g
	// /stats?album=featured&img=1om86jo4rku2r&time=391&w=1620&h=1080&mw=1621&mh=1080&sw=1920&sh=1080&px=2&v=k...g
	// /stats?main=1&sw=1920&sh=1080&px=2&v=k...g
	// /stats?thumb=%7B%2234suxvlfx0lz8%22%3A36704%2C%221z4zoegvmke8n%22%3A36704%2C%223b45tgt52cnms%22%3A36704%2C%221d2ujpqi6nbb4%22%3A36704%2C%221shlwpftv8av4%22%3A36704%7D&sw=1792&sh=1120&px=2&v=1...w
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
