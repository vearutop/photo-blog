package settings

import "context"

type Storage struct {
	WebDAV bool `json:"web_dav" inlineTitle:"Enable WebDAV access to storage." noTitle:"true" title:"Enable WebDAV" description:"Served at http(s)://[this-site-address]/webdav/ URL with admin password."`
}

func (m *Manager) SetStorage(ctx context.Context, value Storage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "storage", value); err != nil {
		return err
	}

	m.storage = value

	return nil
}

func (m *Manager) Storage() Storage {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.storage
}
