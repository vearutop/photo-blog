// Package pixelpeep holds the config shape shared by the pixelpeep CLI
// (cmd/pixelpeep) and the pixelpeep webapp routes: a set of 2-4 images
// compared side by side with synchronized pan/zoom.
package pixelpeep

import "fmt"

// Image is one entry of a pixelpeep set. URL is either a local file path (CLI
// only) or a URL to an already-served image.
type Image struct {
	URL     string  `json:"url"`
	Label   string  `json:"label,omitempty"`
	Zoom    float64 `json:"zoom,omitempty"`    // initial per-image zoom multiplier, default 1.
	OffsetX float64 `json:"offsetX,omitempty"` // initial per-image offset from center, in group-local px.
	OffsetY float64 `json:"offsetY,omitempty"`
}

// Config is the whole pixelpeep set: viewport-level settings plus the images.
type Config struct {
	Height  int     `json:"height,omitempty"` // viewport height in px, default fills the screen.
	Zoom    float64 `json:"zoom,omitempty"`   // initial shared pan/zoom, and what "Reset zoom" returns to.
	OffsetX float64 `json:"offsetX,omitempty"`
	OffsetY float64 `json:"offsetY,omitempty"`
	Items   []Image `json:"items"`
}

// Validate checks the item count is within the 2-4 pixelpeep supports.
func (c Config) Validate() error {
	if len(c.Items) < 2 || len(c.Items) > 4 {
		return fmt.Errorf("need 2-4 images, got %d", len(c.Items))
	}

	return nil
}
