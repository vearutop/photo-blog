package web

import (
	"html/template"
	"io"
)

type Page struct {
	w io.Writer
}

func (o *Page) SetWriter(w io.Writer) {
	o.w = w
}

func (o *Page) Render(tmpl *template.Template, data any) error {
	return tmpl.Execute(o.w, data)
}
