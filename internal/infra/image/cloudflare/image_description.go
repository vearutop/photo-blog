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

// UformGen2 identifies a model, see https://developers.cloudflare.com/workers-ai/models/uform-gen2-qwen-500m.
const UformGen2 = "cf-uform-gen2"

// Following worker code can be deployed on Cloudflare.
/*
export default {
  async fetch(request, env) {
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
        image: [...new Uint8Array(blob)],
        prompt: "Generate a caption for this image",
        max_tokens: 512,
      };

      response[url] = await env.AI.run("@cf/unum/uform-gen2-qwen-500m", inputs);
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

Sample request body:
{"api_key":"foo","images":["https://vearutop.p1cs.art/thumb/1200w/mbu3wmasjobq.jpg"]}

Sample response body:
{"https://vearutop.p1cs.art/thumb/1200w/mbu3wmasjobq.jpg":{"description":"A solitary figure sits on a bench in a park, surrounded by a lush tree with pink flowers. The bench is near a trash can and a laptop, suggesting a moment of rest or contemplation. The park is surrounded by a wall adorned with graffiti, adding a touch of urban artistry. The perspective of the image is from the ground, looking up at the tree, creating a sense of depth and scale. The landmark identifier \"sa_1500\" does not provide additional information about the location of this park."}}
*/

func NewImageDescriber(logger ctxd.Logger, cfg func() ImageWorkerConfig) *ImageDescriber {
	return &ImageDescriber{
		logger: logger,
		cfg:    cfg,
		queue:  make(map[string]func(labels photo.ImageLabel)),
	}
}

type ImageDescriber struct {
	cfg func() ImageWorkerConfig

	logger ctxd.Logger

	mu      sync.Mutex
	ctx     context.Context
	queue   map[string]func(labels photo.ImageLabel)
	pending bool
}

func (ic *ImageDescriber) Describe(ctx context.Context, imgHash uniq.Hash, cb func(labels photo.ImageLabel)) {
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

func (ic *ImageDescriber) doClassify() {
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

func (ic *ImageDescriber) ensureFetch(req *request) {
	defer func() {
		req.Images = req.Images[:0]
	}()

	for {
		err := ic.fetch(req)

		if err == nil {
			return
		}

		if errors.Is(err, status.ResourceExhausted) {
			ic.logger.Warn(ic.ctx, "cf worker resource exhausted, sleeping", "model", UformGen2)

			time.Sleep(time.Minute)

			continue
		}

		ic.logger.Error(ic.ctx, "failed to fetch cf img cls", "images", req.Images, "error", err)
		return
	}
}

func (ic *ImageDescriber) fetch(req *request) (err error) {
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

	var response map[string]struct {
		Description string `json:"description"`
	}

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

		ic.logger.Info(ic.ctx, "cf img description",
			"url", u, "desc", l.Description)

		cb(photo.ImageLabel{
			Model: UformGen2,
			Text:  l.Description,
		})
	}

	return nil
}
