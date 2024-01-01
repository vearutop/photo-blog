package settings

import "context"

type Maps struct {
	Tiles       string `json:"tiles" title:"Tiles" description:"URL to custom map tiles." example:"https://retina-tiles.p.rapidapi.com/local/osm{r}/v1/{z}/{x}/{y}.png?rapidapi-key=YOUR-RAPIDAPI-KEY"`
	Attribution string `json:"attribution" title:"Attribution" description:"Map tiles attribution."`
	Cache       bool   `json:"cache" inlineTitle:"Cache tiles." noTitle:"true" title:"Cache" description:"Enable local cache of map tiles."`
}

func (m *Manager) SetMaps(ctx context.Context, value Maps) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.set(ctx, "maps", value); err != nil {
		return err
	}

	m.maps = value

	return nil
}

func (m *Manager) Maps() Maps {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.maps
}
