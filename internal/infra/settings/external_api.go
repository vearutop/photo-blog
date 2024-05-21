package settings

import (
	"context"

	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
)

type ExternalAPI struct {
	CFImageClassifier cloudflare.ImageWorkerConfig `json:"cf_image_classifier"`
	CFImageDescriber  cloudflare.ImageWorkerConfig `json:"cf_image_describer"`
	FacesRecognizer   faces.RecognizerConfig       `json:"faces_recognizer"`
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

func (m *Manager) CFImageClassifier() cloudflare.ImageWorkerConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.externalAPI.CFImageClassifier
}

func (m *Manager) CFImageDescriber() cloudflare.ImageWorkerConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.externalAPI.CFImageDescriber
}
