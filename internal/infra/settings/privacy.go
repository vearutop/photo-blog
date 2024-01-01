package settings

import "context"

type Privacy struct {
	HideTechDetails bool `json:"hide_tech_details" inlineTitle:"Hide technical details." noTitle:"true" description:"Disables a button that shows EXIF data."`
	HideGeoPosition bool `json:"hide_geo_position" inlineTitle:"Hide geo position." noTitle:"true" description:"Disables location information of images."`
}

func (m *Manager) SetPrivacy(ctx context.Context, value Privacy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "privacy", value); err != nil {
		return err
	}

	m.privacy = value

	return nil
}

func (m *Manager) Privacy() Privacy {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.privacy
}
