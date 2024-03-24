package txt_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/pkg/txt"
)

func TestRenderer_RenderLang(t *testing.T) {
	ctx := context.Background()

	r := txt.NewRenderer()

	md := `
# The Title

Lorem ipsum [foo](https://bar.com).

* bla

:::{lang=fr}
Après deux
:::

:::{lang=ru}
бла бла бла бла
:::

:::{lang=en}
hello, world!
:::

`

	t.Run("ru", func(t *testing.T) {
		s, err := r.RenderLang(txt.WithLanguage(ctx, "ru"), md)
		require.NoError(t, err)
		assert.Equal(t, `<h1 id="the-title">The Title</h1>
<p>Lorem ipsum <a href="https://bar.com">foo</a>.</p>
<ul>
<li>bla</li>
</ul>

<div data-fence="1" lang="ru">
<p>бла бла бла бла</p>
</div>`, strings.TrimSpace(s), s)
	})

	t.Run("en-sanitize", func(t *testing.T) {
		s, err := r.RenderLang(txt.WithLanguage(ctx, "en"), md, func(o *txt.RenderOptions) {
			o.Sanitize = true
		})
		require.NoError(t, err)
		assert.Equal(t, `<h1 id="the-title">The Title</h1>
<p>Lorem ipsum <a href="https://bar.com" rel="nofollow">foo</a>.</p>
<ul>
<li>bla</li>
</ul>


<div lang="en">
<p>hello, world!</p>
</div>`, strings.TrimSpace(s), s)
	})

	t.Run("fr-strip", func(t *testing.T) {
		s, err := r.RenderLang(txt.WithLanguage(ctx, "fr"), md, func(o *txt.RenderOptions) {
			o.StripTags = true
		})
		require.NoError(t, err)
		assert.Equal(t, `The Title
Lorem ipsum foo.

bla


Après deux`, strings.TrimSpace(s), s)
	})
}

func TestRenderer_Render_table(t *testing.T) {
	md := `
# The Title

Lorem ipsum [foo](https://bar.com).

* bla
* bla
* bla

:::{lang=fr}
Après deux ans de silence et de patience, malgré
mes résolutions, je reprends la plume. Lecteur, suspendez votre
jugement sur les raisons qui m’y forcent : vous n’en pouvez juger
qu’après m’avoir lu.
:::

|Name                   |In    |Type   |Examples |
|-----------------------|------|-------|---------|
|t                      |query |String |         |
|notrack                |query |Boolean|         |
|s2s                    |query |Boolean|         |
|gps_adid               |query |String |         |


`

	r := txt.NewRenderer()
	s, err := r.Render(md)

	require.NoError(t, err)

	assert.Equal(t, `<h1 id="the-title">The Title</h1>
<p>Lorem ipsum <a href="https://bar.com">foo</a>.</p>
<ul>
<li>bla</li>
<li>bla</li>
<li>bla</li>
</ul>
<div data-fence="0" lang="fr">
<p>Après deux ans de silence et de patience, malgré<br>
mes résolutions, je reprends la plume. Lecteur, suspendez votre<br>
jugement sur les raisons qui m’y forcent : vous n’en pouvez juger<br>
qu’après m’avoir lu.</p>
</div>
<table>
<thead>
<tr>
<th>Name</th>
<th>In</th>
<th>Type</th>
<th>Examples</th>
</tr>
</thead>
<tbody>
<tr>
<td>t</td>
<td>query</td>
<td>String</td>
<td></td>
</tr>
<tr>
<td>notrack</td>
<td>query</td>
<td>Boolean</td>
<td></td>
</tr>
<tr>
<td>s2s</td>
<td>query</td>
<td>Boolean</td>
<td></td>
</tr>
<tr>
<td>gps_adid</td>
<td>query</td>
<td>String</td>
<td></td>
</tr>
</tbody>
</table>`, strings.TrimSpace(s), s)
}

func TestRenderer_Render_stripTags(t *testing.T) {
	r := txt.NewRenderer()
	s, err := r.Render(`Devil's Bridge`, func(o *txt.RenderOptions) {
		o.StripTags = true
	})

	require.NoError(t, err)
	assert.Equal(t, "Devil's Bridge", s)
}

func TestRenderer_Render_codeBlock(t *testing.T) {
	ctx := context.Background()
	ctx = txt.WithLanguage(ctx, "ru")

	r := txt.NewRenderer()
	s := r.MustRenderLang(ctx, "```\n\n:::{lang=en}\n\nfoo\n\n:::\n\n:::{lang=ru}\n\nбар\n\n:::\n\n```\n\n:::{lang=en}\n\nbaz\n\n:::\n")

	assert.Equal(t, `<pre><code>
:::{lang=en}

foo

:::

:::{lang=ru}

бар

:::

</code></pre>`, s)
}

func TestRenderer_Render_multiLine(t *testing.T) {
	ctx := context.Background()
	ctx = txt.WithLanguage(ctx, "en")

	r := txt.NewRenderer()
	s := r.MustRenderLang(ctx, `
line1
line2
line3
`)

	assert.Equal(t, `<p>line1<br/>
line2<br/>
line3</p>`, s)
}

// BenchmarkRenderer_MustRenderLang-16    	   37148	     29364 ns/op	   27071 B/op	     200 allocs/op.
func BenchmarkRenderer_MustRenderLang(b *testing.B) {
	ctx := context.Background()

	r := txt.NewRenderer()

	md := `
# The Title

Lorem ipsum [foo](https://bar.com).

* bla

:::{lang=fr}
Après deux
:::

:::{lang=ru}
бла бла бла бла
:::

:::{lang=en}
hello, world!
:::

`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := r.RenderLang(txt.WithLanguage(ctx, "ru"), md)
		if err != nil {
			b.Fail()
		}
	}
}
