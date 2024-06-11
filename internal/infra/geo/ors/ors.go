// Package ors implements a client to openrouteservice.org.
//
// See also https://openrouteservice.org/dev/#/api-docs/.
package ors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bool64/ctxd"
)

type Client struct {
	logger    *ctxd.Logger
	deps      Deps
	cfg       func() Config
	APIKey    string
	Transport http.RoundTripper
	mu        sync.Mutex
}

type Config struct {
	APIKey string `json:"api_key"`
}

type Deps interface {
	CtxdLogger() ctxd.Logger
}

func NewORS(deps Deps, cfg func() Config) *Client {
	return &Client{
		deps: deps,
		cfg:  cfg,
	}
}

type ReverseResp struct {
	Geocoding struct {
		Version     string `json:"version"`
		Attribution string `json:"attribution"`
		Query       struct {
			Size              int     `json:"size"`
			Private           bool    `json:"private"`
			PointLat          float64 `json:"point.lat"`
			PointLon          float64 `json:"point.lon"`
			BoundaryCircleLat float64 `json:"boundary.circle.lat"`
			BoundaryCircleLon float64 `json:"boundary.circle.lon"`
			Lang              struct {
				Name      string `json:"name"`
				Iso6391   string `json:"iso6391"`
				Iso6393   string `json:"iso6393"`
				Via       string `json:"via"`
				Defaulted bool   `json:"defaulted"`
			} `json:"lang"`
			QuerySize int `json:"querySize"`
		} `json:"query"`
		Engine struct {
			Name    string `json:"name"`
			Author  string `json:"author"`
			Version string `json:"version"`
		} `json:"engine"`
		Timestamp int64 `json:"timestamp"`
	} `json:"geocoding"`
	Type     string `json:"type"`
	Features []struct {
		Type     string `json:"type"`
		Geometry struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			Id               string  `json:"id"`
			Gid              string  `json:"gid"`
			Layer            string  `json:"layer"`
			Source           string  `json:"source"`
			SourceId         string  `json:"source_id"`
			Name             string  `json:"name"`
			Confidence       float64 `json:"confidence"`
			Distance         float64 `json:"distance"`
			Accuracy         string  `json:"accuracy"`
			Country          string  `json:"country"`
			CountryGid       string  `json:"country_gid"`
			CountryA         string  `json:"country_a"`
			Macroregion      string  `json:"macroregion"`
			MacroregionGid   string  `json:"macroregion_gid"`
			MacroregionA     string  `json:"macroregion_a"`
			Region           string  `json:"region"`
			RegionGid        string  `json:"region_gid"`
			RegionA          string  `json:"region_a"`
			Localadmin       string  `json:"localadmin"`
			LocaladminGid    string  `json:"localadmin_gid"`
			Locality         string  `json:"locality"`
			LocalityGid      string  `json:"locality_gid"`
			Borough          string  `json:"borough"`
			BoroughGid       string  `json:"borough_gid"`
			Neighbourhood    string  `json:"neighbourhood"`
			NeighbourhoodGid string  `json:"neighbourhood_gid"`
			Continent        string  `json:"continent"`
			ContinentGid     string  `json:"continent_gid"`
			Label            string  `json:"label"`
			Addendum         struct {
				Osm struct {
					Wheelchair string `json:"wheelchair"`
					Wikidata   string `json:"wikidata,omitempty"`
					Website    string `json:"website,omitempty"`
				} `json:"osm"`
			} `json:"addendum,omitempty"`
			Street     string `json:"street,omitempty"`
			Postalcode string `json:"postalcode,omitempty"`
		} `json:"properties"`
		Bbox []float64 `json:"bbox,omitempty"`
	} `json:"features"`
	Bbox []float64 `json:"bbox"`
}

func (o *Client) ReverseGeocode(ctx context.Context, lat, lon float64) (string, error) {
	apiKey := o.cfg().APIKey
	if apiKey == "" {
		return "", errors.New("no api key")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	tr := o.Transport

	if tr == nil {
		tr = http.DefaultTransport
	}

	u := fmt.Sprintf("https://api.openrouteservice.org/geocode/reverse?api_key=%s&point.lon=%f1&point.lat=%f", apiKey, lon, lat)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("new ors reverse req: %w", err)
	}

	var body []byte

loop:
	for {
		resp, err := tr.RoundTrip(req)
		if err != nil {
			return "", fmt.Errorf("ors send: %w", err)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("ors read response: %w", err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			break loop
		case http.StatusTooManyRequests:
			o.deps.CtxdLogger().Warn(ctx, "ors sleeping for too many requests")
			time.Sleep(time.Minute)
			continue

		case http.StatusForbidden:
			o.deps.CtxdLogger().Warn(ctx, "ors sleeping for forbidden")
			time.Sleep(time.Hour)
			continue
		default:
			return "", fmt.Errorf("ors reverse status: %s", resp.Status)
		}
	}

	var data ReverseResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("ors parse response: %w", err)
	}

	o.deps.CtxdLogger().Info(ctx, "reverse geocode",
		"lat", lat,
		"lon", lon,
		"response", data)

	if len(data.Features) > 0 {
		return data.Features[0].Properties.Label, nil
	}

	return "", nil
}
