package nethttp_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
	"time"

	"github.com/bool64/brick/runtime"
	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/greeting"
	"github.com/vearutop/photo-blog/internal/infra"
	"github.com/vearutop/photo-blog/internal/infra/nethttp"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

func Benchmark_hello(b *testing.B) {
	log.SetOutput(io.Discard)

	cfg := service.Config{}
	cfg.Initialized = true
	cfg.Log.Output = io.Discard
	cfg.ShutdownTimeout = time.Second
	l, err := infra.NewServiceLocator(cfg)
	require.NoError(b, err)

	l.GreetingMakerProvider = &greeting.SimpleMaker{}

	r := nethttp.NewRouter(l)

	httptestbench.ServeHTTP(b, 50, r,
		func(i int) *http.Request {
			req, err := http.NewRequest(http.MethodGet, "/hello?name=Jack&locale=en-US", nil)
			if err != nil {
				b.Fatal(err)
			}

			return req
		},
		func(i int, resp *httptest.ResponseRecorder) bool {
			return resp.Code == http.StatusOK
		},
	)

	b.StopTimer()
	b.ReportMetric(float64(runtime.StableHeapInUse())/float64(1024*1024), "MB/inuse")
	l.Shutdown()
	require.NoError(b, <-l.Wait())
}

func TestOlolo(t *testing.T) {
	type Secret struct {
		Name     string
		Registry string
		Username string
		Pass     string
		Email    string
	}

	type Data struct {
		ImageCredentials []Secret
	}

	tmpl := template.New("foo")
	tmpl, err := tmpl.Parse(`
{{- define "imagePullSecret" }}
{{- range $secret := .imageCredentials }}
  {{ $secret.Name }}
{{- end }}
{{- end }}
`)
	require.NoError(t, err)

	data := Data{
		ImageCredentials: []Secret{
			{
				Name: "foo",
			},
			{
				Name: "bar",
			},
		},
	}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, tmpl.Execute(buf, data))

	fmt.Println(buf.String())
}
