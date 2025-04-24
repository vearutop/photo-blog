package settings

import (
	"context"
)

type Indexing struct {
	Faces                bool `json:"faces" inlineTitle:"Recognize faces." noTitle:"true" title:"Faces" description:"Enable faces indexing."`
	CFClassification     bool `json:"cf_classification" inlineTitle:"ResNet50 labels." noTitle:"true" title:"ResNet50" description:"Image labels."`
	CFDescription        bool `json:"cf_description" inlineTitle:"Legacy CF image description." noTitle:"true" title:"CF Description"`
	GeoLabel             bool `json:"geo_label" inlineTitle:"Reverse geo tag." noTitle:"true"`
	LLMDescription       bool `json:"llm_description" inlineTitle:"Prompt LLM for image description." noTitle:"true"`
	Phash                bool `json:"phash" inlineTitle:"Calculate perception hash." noTitle:"true"`
	SharpnessV0          bool `json:"sharpness_v0" inlineTitle:"Calculate sharpness (legacy)." noTitle:"true"`
	Skip2400wThumb       bool `json:"skip_2400_w_thumb" inlineTitle:"Skip 2400w thumbnail." noTitle:"true"`
	TemporaryLargeThumbs bool `json:"temporary_large_thumbs" inlineTitle:"Temporary large thumbnail." noTitle:"true" description:"Do not persist 1200w, 2400w thumbs to save space."`
}

func (m *Manager) SetIndexing(ctx context.Context, value Indexing) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "indexing", value); err != nil {
		return err
	}

	m.indexing = value

	return nil
}

func (m *Manager) Indexing() Indexing {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.indexing
}
