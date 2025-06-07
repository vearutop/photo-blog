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

var TableTemplate = MustParseTemplate("stats/table.html")

func MustParseTemplate(fileName string) *template.Template {
	tpl, err := Assets.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	return tmpl
}

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
