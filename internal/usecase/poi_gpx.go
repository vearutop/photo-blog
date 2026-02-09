package usecase

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/tkrajina/gpxgo/gpx"
)

type dlImagesPoiGpxInput struct {
	request.EmbeddedSetter
	Name string `path:"name"`
}

// ShowAlbum creates use case interactor to show album.
func DownloadImagesPoiGpx(deps getAlbumImagesDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in dlImagesPoiGpxInput, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		cont, err := getAlbumContents(ctx, deps, imagesFilter{albumName: in.Name}, false)
		if err != nil {
			return fmt.Errorf("get album contents: %w", err)
		}

		gpxDoc := gpx.GPX{}

		for _, i := range cont.Images {
			if i.Gps != nil {
				g := i.Gps

				p := gpx.GPXPoint{}
				if i.Description != "" {
					p.Description += i.Description + "\n\n"
				}
				p.Description += fmt.Sprintf(`<img src="https://p1cs.1337.cx/thumb/300w/%s.jpg" />`, i.Hash)
				p.Latitude = g.Latitude
				p.Longitude = g.Longitude

				gpxDoc.AppendWaypoint(&p)
			}
		}

		x, err := gpxDoc.ToXml(gpx.ToXmlParams{
			Indent: true,
		})
		if err != nil {
			return err
		}

		http.ServeContent(rw, in.Request(), "photos-"+cont.Album.Name+".gpx", time.Now(), bytes.NewReader(x))

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
