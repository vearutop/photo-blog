package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const ResNet50 = "cf-resnet-50"

// Following worker code can be deployed on Cloudflare.
/*
import { Ai } from './vendor/@cloudflare/ai.js';

export default {
  async fetch(request, env) {
    const ai = new Ai(env.AI);
    const req = await request.json();
    const images = req.images;

    if (req.api_key !== "foo") {
      // Incorrect key supplied. Reject the request.
      return new Response("Sorry, you have supplied an invalid key.", {
        status: 403,
      });
    }


    var response = {};

    const process = async function(url) {
      const imageResponse = await fetch(url);
      const blob = await imageResponse.arrayBuffer();

      const inputs = {
        image: [...new Uint8Array(blob)]
      };

      response[url] = await ai.run('@cf/microsoft/resnet-50', inputs);
    }

    var promises = [];

    for (var i in images) {
      const url = images[i];
      promises.push(process(url));
    }

    for (const i in promises) {
      await promises[i]
    }

    return Response.json(response);
  }
};
*/

type ImageWorkerConfig struct {
	URL              string `json:"url" example:"https://image-classification-soft-rain-fb0b.nanopeni.workers.dev/"`
	ImageURLTemplate string `json:"image_url_template" example:"https://vearutop.p1cs.art/thumb/1200w/%s.jpg"`
	APIKey           string `json:"api_key"`
	BatchSize        int    `json:"batch_size" default:"10"`
}

func NewImageClassifier(logger ctxd.Logger, cfg func() ImageWorkerConfig) *ImageClassifier {
	return &ImageClassifier{
		logger: logger,
		cfg:    cfg,
		queue:  make(map[string]func(labels []photo.ImageLabel)),
	}
}

type ImageClassifier struct {
	cfg func() ImageWorkerConfig

	logger ctxd.Logger

	mu      sync.Mutex
	ctx     context.Context
	queue   map[string]func(labels []photo.ImageLabel)
	pending bool
}

func (ic *ImageClassifier) Classify(ctx context.Context, imgHash uniq.Hash, cb func(labels []photo.ImageLabel)) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	cfg := ic.cfg()

	if cfg.ImageURLTemplate == "" {
		return
	}

	u := fmt.Sprintf(cfg.ImageURLTemplate, imgHash.String())
	ic.queue[u] = cb
	ic.ctx = ctx

	if !ic.pending {
		ic.pending = true
		time.AfterFunc(time.Second, func() {
			defer func() {
				if r := recover(); r != nil {
					ic.logger.Error(context.Background(), "panic recovered", "value", r)
				}
			}()
			ic.doClassify()
		})
	}
}

type request struct {
	APIKey string   `json:"api_key"`
	Images []string `json:"images"`
}

type label struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

func (ic *ImageClassifier) doClassify() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	//{"api_key":"foo","images":[
	//"https://vearutop.p1cs.art/thumb/1200w/7611nbymdl68.jpg",
	//"https://vearutop.p1cs.art/thumb/1200w/211s0s5scopys.jpg",
	//"https://vearutop.p1cs.art/thumb/1200w/6hsfgwt2k65m.jpg"
	//]}

	cfg := ic.cfg()

	req := request{
		APIKey: cfg.APIKey,
	}

	for u := range ic.queue {
		req.Images = append(req.Images, u)

		if len(req.Images) >= cfg.BatchSize {
			ic.ensureFetch(&req)
		}
	}

	if len(req.Images) > 0 {
		ic.ensureFetch(&req)
	}

	ic.pending = false
}

func (ic *ImageClassifier) ensureFetch(req *request) {
	defer func() {
		req.Images = req.Images[:0]
	}()

	for {
		err := ic.fetch(req)

		if err == nil {
			return
		}

		if errors.Is(err, status.ResourceExhausted) {
			ic.logger.Warn(ic.ctx, "cf worker resource exhausted, sleeping", "model", ResNet50)

			time.Sleep(time.Minute)

			continue
		}

		ic.logger.Error(ic.ctx, "failed to fetch cf img cls", "images", req.Images, "error", err)
		return
	}
}

func (ic *ImageClassifier) fetch(req *request) (err error) {
	_, done := opencensus.AddSpan(context.Background())
	defer done(&err)

	cfg := ic.cfg()

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	res, err := http.Post(cfg.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := res.Body.Close(); err != nil {
		return err
	}

	if res.StatusCode == http.StatusServiceUnavailable && bytes.Contains(resp, []byte("Worker exceeded resource limits")) {
		return status.ResourceExhausted
	}

	if res.StatusCode != http.StatusOK {
		return ctxd.NewError(ic.ctx, "unexpected status code",
			"code", res.StatusCode,
			"resp", string(resp),
			"header", res.Header,
		)
	}

	var response map[string][]label

	if err := json.Unmarshal(resp, &response); err != nil {
		return err
	}

	for u, l := range response {
		cb := ic.queue[u]
		delete(ic.queue, u)

		if cb == nil {
			ic.logger.Warn(ic.ctx, "empty callback for cf cls", "url", u)
			continue
		}

		labels := make([]photo.ImageLabel, len(l))

		for i, lbl := range l {
			labels[i] = photo.ImageLabel{
				Model: ResNet50,
				Text:  lbl.Label,
				Score: lbl.Score,
			}
		}

		ic.logger.Info(ic.ctx, "cf img classification",
			"url", u, "labels", labels)

		cb(labels)
	}

	return nil
}
