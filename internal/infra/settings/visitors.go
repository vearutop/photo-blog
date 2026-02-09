package settings

import "context"

type Visitors struct {
	Tag             bool     `json:"tag" inlineTitle:"Tag unique visitors with cookies." noTitle:"true"`
	AccessLog       bool     `json:"access_log" inlineTitle:"Enable access log." noTitle:"true"`
	IgnoreReferrers []string `json:"ignore_referrers,omitempty" title:"Ignore referrers" description:"List of referrer URL prefixes to ignore."`
	TrustedProxies  []string `json:"trusted_proxies,omitempty" title:"Trusted proxies" description:"List of IP addresses of trusted proxies."`
	CityDB          string   `json:"city_db" title:"City location DB" description:"Local path to DB, download and decompress from https://github.com/vearutop/ipinfo/releases/download/index/city-loc-lite.bin.zst."`
	ASNBotDB        string   `json:"asn_bot_db" title:"ASN bot DB" description:"Local path to DB, download and decompress from https://github.com/vearutop/ipinfo/releases/download/index/asn-bot.bin.zst."`
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
