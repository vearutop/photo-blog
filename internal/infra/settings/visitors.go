package settings

import "context"

type Visitors struct {
	Tag       bool `json:"tag" inlineTitle:"Tag unique visitors with cookies." noTitle:"true"`
	AccessLog bool `json:"access_log" inlineTitle:"Enable access log." noTitle:"true"`
}

func (m *Manager) SetVisitors(ctx context.Context, value Visitors) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "visitors", value); err != nil {
		return err
	}

	m.visitors = value

	return nil
}

func (m *Manager) Visitors() Visitors {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.visitors
}
