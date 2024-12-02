package settings

import (
	"context"
	"fmt"

	"golang.org/x/text/language"
)

type Appearance struct {
	SiteTitle   string `json:"site_title" title:"Title" formType:"textarea" description:"The title of this site."`
	SiteFavicon string `json:"site_favicon" title:"Link to favicon" description:"Defaults to /static/favicon.png, you can upload your own and use, for example, /site/favicon.png."`
	SiteHead    string `json:"site_head" title:"HTML Head" formType:"textarea" description:"Injected at the end of page &lt;html&gt;&lt;head&gt; element."`
	SiteHeader  string `json:"site_header" title:"Header" formType:"textarea" description:"Injected at page start."`
	SiteFooter  string `json:"site_footer" title:"Footer" formType:"textarea" description:"Injected at page end."`

	FeaturedAlbumName string `split_words:"true" default:"featured" json:"featured_album_name" title:"Featured album name" description:"The name of an album to show on the main page."`

	Languages    []string `json:"languages" title:"Languages" description:"Supported content languages."`
	ThumbBaseURL string   `json:"thumb_base_url" title:"Thumbnails Base URL" description:"Optional custom URL for thumbnails." example:"https://example.org/thumb"`

	languageMatcher language.Matcher
}

func (a Appearance) LanguageMatcher() (language.Matcher, []string) {
	return a.languageMatcher, a.Languages
}

func (a *Appearance) change() error {
	if len(a.Languages) <= 1 {
		return nil
	}

	languages := a.Languages
	var tags []language.Tag
	for _, l := range languages {
		t, err := language.Parse(l)
		if err != nil {
			return fmt.Errorf("parse language %s: %w", l, err)
		}
		tags = append(tags, t)
	}
	matcher := language.NewMatcher(tags)

	a.languageMatcher = matcher

	return nil
}

func (m *Manager) SetAppearance(ctx context.Context, value Appearance) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := m.appearance
	defer func() {
		if err != nil {
			m.appearance = old
		}
	}()

	m.appearance = value

	if err = m.appearance.change(); err != nil {
		return err
	}

	if err = m.set(ctx, "appearance", value); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Appearance() Appearance {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.appearance
}
