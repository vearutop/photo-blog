package settings

import (
	"context"
	"fmt"

	"golang.org/x/text/language"
)

type Appearance struct {
	SiteTitle         string   `json:"site_title" title:"Title" description:"The title of this site."`
	FeaturedAlbumName string   `split_words:"true" default:"featured" json:"featured_album_name" title:"Featured album name" description:"The name of an album to show on the main page."`
	Languages         []string `json:"languages" title:"Languages" description:"Supported content languages."`

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
