package web

import (
	"html/template"

	"github.com/swaggest/rest/response"
)

type Page struct {
	response.EmbeddedSetter
}

func (o *Page) Render(tmpl *template.Template, data any) error {
	return tmpl.Execute(o.ResponseWriter(), data)
}
