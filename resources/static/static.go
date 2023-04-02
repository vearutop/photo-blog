// Package static provides embedded static assets.
package static

import (
	"embed"
	"html/template"
)

// Assets provides embedded static assets for web application.
//
//go:embed *
var Assets embed.FS

func Template(fileName string) (*template.Template, error) {
	tpl, err := Assets.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
