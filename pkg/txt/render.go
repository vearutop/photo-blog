package txt

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

type Renderer struct {
	sanitizer *bluemonday.Policy
	strict    *bluemonday.Policy
	md        goldmark.Markdown
}

func NewRenderer() *Renderer {
	return &Renderer{
		sanitizer: bluemonday.UGCPolicy(),
		strict:    bluemonday.StrictPolicy(),
		md: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				&fences.Extender{},
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				// gmhtml.WithHardWraps(),
				gmhtml.WithUnsafe(),
			),
		),
	}
}

type langCtxKey struct{}

func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, langCtxKey{}, lang)
}

func Language(ctx context.Context) string {
	if lang, ok := ctx.Value(langCtxKey{}).(string); ok {
		return lang
	}

	return "en"
}

type RenderOptions struct {
	Lang string

	// Sanitize strips malicious code from untrusted HTML, not needed for trusted input.
	Sanitize bool

	// StripTags strips all HTML tags.
	StripTags bool
}

func (r *Renderer) Render(source string, opts ...func(o *RenderOptions)) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := r.md.Convert([]byte(source), buf)
	if err != nil {
		return "", err
	}

	o := RenderOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	res := buf.String()

	if o.Lang != "" {
		res, err = r.filterLang(buf.String(), o.Lang)
		if err != nil {
			return "", fmt.Errorf("filter lang: %w", err)
		}

	}

	if o.Sanitize {
		res = r.sanitizer.Sanitize(res)
	}

	if o.StripTags {
		res = r.strict.Sanitize(res)
		res = html.UnescapeString(res)
	}

	return strings.TrimSpace(res), nil
}

func (r *Renderer) MustRenderLang(ctx context.Context, source string, opts ...func(o *RenderOptions)) string {
	s, err := r.Render(source, func(o *RenderOptions) {
		o.Lang = Language(ctx)

		for _, opt := range opts {
			opt(o)
		}
	})
	if err != nil {
		panic(err)
	}

	return s
}

func (r *Renderer) RenderLang(ctx context.Context, source string, opts ...func(o *RenderOptions)) (string, error) {
	return r.Render(source, func(o *RenderOptions) {
		o.Lang = Language(ctx)

		for _, opt := range opts {
			opt(o)
		}
	})
}

func (r *Renderer) filterLang(htmlDoc string, lang string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlDoc))
	if err != nil {
		return "", fmt.Errorf("parse html %q: %w", htmlDoc, err)
	}

	if doc == nil || doc.FirstChild == nil || doc.FirstChild.LastChild == nil {
		return htmlDoc, nil
	}

	var remove []*html.Node

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "lang" {
					if a.Val != lang {
						remove = append(remove, n)
					}

					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	for _, r := range remove {
		r.Parent.RemoveChild(r)
	}

	render := bytes.NewBuffer(nil)
	err = html.Render(render, doc.FirstChild.LastChild)
	if err != nil {
		return "", fmt.Errorf("render html %q: %w", htmlDoc, err)
	}

	s := render.String()
	s = strings.TrimPrefix(s, "<body>")
	s = strings.TrimSuffix(s, "</body>")

	return s, nil
}

func (r *Renderer) TxtRenderer() *Renderer {
	return r
}
