package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bool64/ctxd"
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

type ImageClassifierConfig struct {
	URL              string `json:"url" example:"https://image-classification-soft-rain-fb0b.nanopeni.workers.dev/"`
	ImageURLTemplate string `json:"image_url_template" example:"https://vearutop.p1cs.art/thumb/1200w/%s.jpg"`
	APIKey           string `json:"api_key"`
	BatchSize        int    `json:"batch_size" default:"10"`
}

func NewImageClassifier(logger ctxd.Logger, cfg func() ImageClassifierConfig) *ImageClassifier {
	return &ImageClassifier{
		logger: logger,
		cfg:    cfg,
		queue:  make(map[string]func(labels []photo.ImageLabel)),
	}
}

type ImageClassifier struct {
	cfg func() ImageClassifierConfig

	logger ctxd.Logger

	mu      sync.Mutex
	queue   map[string]func(labels []photo.ImageLabel)
	pending bool
}

func (ic *ImageClassifier) Classify(imgHash uniq.Hash, cb func(labels []photo.ImageLabel)) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	cfg := ic.cfg()

	u := fmt.Sprintf(cfg.ImageURLTemplate, imgHash.String())
	ic.queue[u] = cb

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
			if err := ic.fetch(&req); err != nil {
				ic.logger.Error(context.Background(), "failed to fetch cf img cls", "images", req.Images, "error", err)
			}

			req.Images = req.Images[:0]
		}
	}

	if len(req.Images) > 0 {
		if err := ic.fetch(&req); err != nil {
			println("ERR:", err.Error())
		}
	}

	ic.pending = false
}

func (ic *ImageClassifier) fetch(req *request) error {
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

	var response map[string][]label

	if err := json.Unmarshal(resp, &response); err != nil {
		return err
	}

	for u, l := range response {
		cb := ic.queue[u]
		delete(ic.queue, u)

		if cb == nil {
			ic.logger.Warn(context.Background(), "empty callback for cf cls", "url", u)
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

		ic.logger.Info(context.Background(), "cf img classification",
			"url", u, "labels", labels)

		cb(labels)
	}

	return nil
}
