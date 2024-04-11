package settings

import (
	"context"

	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
)

type ExternalAPI struct {
	CFImageClassifier cloudflare.ImageClassifierConfig `json:"cf_image_classifier"`
}

func (m *Manager) SetExternalAPI(ctx context.Context, value ExternalAPI) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "external_api", value); err != nil {
		return err
	}

	m.externalAPI = value

	return nil
}

func (m *Manager) ExternalAPI() ExternalAPI {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.externalAPI
}

func (m *Manager) CFImageClassifier() cloudflare.ImageClassifierConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.externalAPI.CFImageClassifier
}
