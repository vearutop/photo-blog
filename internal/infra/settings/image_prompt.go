package settings

import (
	"context"

	"github.com/vearutop/image-prompt/multi"
)

func (m *Manager) SetImagePrompt(ctx context.Context, value multi.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "image_prompt", value); err != nil {
		return err
	}

	m.imagePrompt = value

	return nil
}

func (m *Manager) ImagePrompt() multi.Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.imagePrompt
}
