package settings

import (
	"context"
)

type Security struct {
	PassHash string `json:"pass_hash"`
	PassSalt string `json:"pass_salt"`
}

func (s Security) Disabled() bool {
	return s.PassHash == "" && s.PassSalt == ""
}

func (m *Manager) SetSecurity(ctx context.Context, value Security) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "security", value); err != nil {
		return err
	}

	m.security = value

	return nil
}

func (m *Manager) Security() Security {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.security
}
