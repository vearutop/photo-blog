package settings

import "context"

type Privacy struct {
	HideTechDetails   bool `json:"hide_tech_details" inlineTitle:"Hide technical details." noTitle:"true" description:"Disables a button that shows EXIF data."`
	HideGeoPosition   bool `json:"hide_geo_position" inlineTitle:"Hide geo position." noTitle:"true" description:"Disables location information of images."`
	HideOriginal      bool `json:"hide_original" inlineTitle:"Hide original images." noTitle:"true" description:"Only shows reduced size images with stripped meta tags (except for 360 panoramas)."`
	HideBatchDownload bool `json:"hide_batch_download" inlineTitle:"Hide batch download." noTitle:"true" description:"Do not allow downloading album images in a ZIP archive."`
	HideLoginButton   bool `json:"hide_login_button" inlineTitle:"Hide login button." noTitle:"true" description:"To not confuse guests, you can remove login link from the bottom of home page and bookmark its destination ('/login') instead."`
	PublicHelp        bool `json:"public_help" inlineTitle:"Publicly show help page." noTitle:"true" description:"Disables auth requirement for '/help'."`
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
