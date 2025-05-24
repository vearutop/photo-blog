package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/topic"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

type indexRemoteDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
	PhotoImageFinder() uniq.Finder[photo.Image]

	PhotoExifEnsurer() uniq.Ensurer[photo.Exif]
	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]

	QueueBroker() *qlite.Broker
}

// IndexRemote creates use case interactor to index remote directory of photos and ensure image relations.
func IndexRemote(deps indexRemoteDeps) usecase.Interactor {
	type indexRemoteInput struct {
		BaseURL string   `json:"base_url"`
		Lists   []string `json:"lists" description:"URL Paths of JSON list files."`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in indexRemoteInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "index_remote", 1)

		for _, u := range in.Lists {
			lu, err := url.JoinPath(in.BaseURL, u)
			if err != nil {
				return err
			}

			if err := deps.QueueBroker().Publish(ctx, topic.IndexRemote, lu); err != nil {
				deps.CtxdLogger().Error(ctx, "error publishing index remote", "error", err)
				return err
			}
		}

		return nil
	})

	u.SetDescription("Index http-remote directories of photos.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

func DoIndexRemote(deps indexRemoteDeps) usecase.IOInteractorOf[string, struct{}] {
	u := usecase.NewInteractor(func(ctx context.Context, listURL string, output *struct{}) error {
		listResp, err := http.Get(listURL)
		if err != nil {
			return fmt.Errorf("getting JSON list: %w", err)
		}

		defer func() {
			if err := listResp.Body.Close(); err != nil {
				deps.CtxdLogger().Warn(ctx, "error closing response body", "error", err)
			}
		}()

		j, err := io.ReadAll(listResp.Body)
		if err != nil {
			return fmt.Errorf("reading JSON list: %w", err)
		}

		var list []image.Data
		if err := json.Unmarshal(j, &list); err != nil {
			return fmt.Errorf("unmarshalling JSON list: %w", err)
		}

		baseURL, _ := path.Split(listURL)

		for _, l := range list {
			newImg := l.Image
			img, err := deps.PhotoImageFinder().FindByHash(ctx, newImg.Hash)
			if errors.Is(err, status.NotFound) {
				deps.CtxdLogger().Warn(ctx, "remote image not found locally", "image", newImg)

				continue
			}

			httpSource := baseURL + newImg.Path

			alreadyLinked := false
			for _, s := range img.Settings.HTTPSources {
				if s == httpSource {
					alreadyLinked = true
					break
				}
			}

			if alreadyLinked {
				continue
			}

			img.Settings.HTTPSources = append([]string{httpSource}, img.Settings.HTTPSources...)
			img.Settings.UpdatedAt = time.Now()

			if _, err := deps.PhotoImageEnsurer().Ensure(ctx, img); err != nil {
				deps.CtxdLogger().Error(ctx, "error ensuring photo image", "error", err)
				continue
			}
		}

		return nil
	})

	u.SetName(topic.IndexRemote)

	return u
}
