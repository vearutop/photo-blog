package usecase

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"golang.org/x/net/html"
)

type ogDeps interface {
	CtxdLogger() ctxd.Logger
}

func OG(deps ogDeps) usecase.Interactor {
	type req struct {
		request.EmbeddedSetter
	}

	return usecase.NewInteractor(func(ctx context.Context, input req, output *response.EmbeddedSetter) error {
		rw := output.ResponseWriter()

		rw.Header().Set("Content-Type", "text/html")

		hd := ""
		for k, v := range input.Request().Header {
			hd += "<p>" + k + ": " + v[0] + "</p>"
		}

		deps.CtxdLogger().Info(ctx, "og page requested",
			"header", input.Request().Header,
			"requestUri", input.Request().RequestURI,
		)

		_, _ = rw.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Bla bla</title>

    <meta property="og:title" content="User-Agent: ` + html.EscapeString(input.Request().Header.Get("User-Agent")) + `"/>
    <meta property="og:description" content="URL: ` + html.EscapeString(input.Request().RequestURI) + `, IP: ` + input.Request().Header.Get("X-Forwarded-For") + `"/>
    <meta property="og:type" content="website"/>
</head>

<p>User-Agent:  ` + html.EscapeString(input.Request().Header.Get("User-Agent")) + `</p>
<p>IP: ` + input.Request().Header.Get("X-Forwarded-For") + `</p>
<p>URL:  ` + html.EscapeString(input.Request().RequestURI) + `</p>
<p>Referer:  ` + html.EscapeString(input.Request().Header.Get("Referer")) + `</p>

Headers: ` + hd + `

</html>
`))

		return nil
	})
}
