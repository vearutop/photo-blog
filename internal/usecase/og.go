package usecase

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/pkg/webstats"
	"golang.org/x/net/html"
)

func OG(deps *service.Locator) usecase.Interactor {
	type req struct {
		request.EmbeddedSetter
		TargetURL string `query:"target_url"`
	}

	return usecase.NewInteractor(func(ctx context.Context, input req, output *response.EmbeddedSetter) error {
		reqh := input.Request().Header

		deps.CtxdLogger().Info(ctx, "og page requested",
			"header", reqh,
			"requestUri", input.Request().RequestURI,
		)

		rw := output.ResponseWriter()

		if input.TargetURL != "" {
			http.Redirect(rw, input.Request(), input.TargetURL, http.StatusMovedPermanently)
		}

		rw.Header().Set("Content-Type", "text/html")

		isBot := webstats.IsBot(reqh.Get("User-Agent"))
		botName := ""
		city := ""

		ip := reqh.Get("X-Forwarded-For")
		if ip != "" {
			// Trusted proxies are removed from the IP chain.
			for _, p := range deps.Settings().Visitors().TrustedProxies {
				if strings.HasSuffix(ip, ", "+p) {
					ip = strings.TrimSuffix(ip, ", "+p)
					break
				}
			}

			reqh.Set("X-Forwarded-For", ip)

			if strings.Contains(ip, ", ") {
				ip = ip[strings.LastIndex(ip, ", ")+2:]
			}
		}

		if ip != "" && deps.ASNBot != nil {
			if botName, _ = deps.ASNBot.SafeLookupIP(net.ParseIP(ip)); botName != "" {
				isBot = true
			}
		}

		if ip != "" && deps.CityLoc != nil {
			city, _ = deps.CityLoc.SafeLookupIP(net.ParseIP(ip))
		}

		hd := ""
		for k, v := range reqh {
			hd += "<p>" + k + ": " + v[0] + "</p>"
		}

		_, _ = rw.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <title>OpenGraph Echo</title>

    <meta property="og:title" content="User-Agent: ` + html.EscapeString(reqh.Get("User-Agent")) + `"/>
    <meta property="og:description" content="URL: ` + html.EscapeString(input.Request().RequestURI) +
			`, IP: ` + reqh.Get("X-Forwarded-For") +
			`, Accept-Language: ` + reqh.Get("Accept-Language") +
			`"/>
    <meta property="og:type" content="website"/>
</head>

<p>User-Agent:  ` + html.EscapeString(reqh.Get("User-Agent")) + `</p>
<p>IP: ` + reqh.Get("X-Forwarded-For") + `</p>
<p>URL:  ` + html.EscapeString(input.Request().RequestURI) + `</p>
<p>Referer:  ` + html.EscapeString(reqh.Get("Referer")) + `</p>
<p>IsBot:  ` + strconv.FormatBool(isBot) + " " + botName + `</p>
<p>Loc:  ` + city + `</p>

Headers: ` + hd + `

</html>
`))

		return nil
	})
}
