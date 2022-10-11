package nethttp_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bool64/brick-starter-kit/internal/domain/greeting"
	"github.com/bool64/brick-starter-kit/internal/infra"
	"github.com/bool64/brick-starter-kit/internal/infra/nethttp"
	"github.com/bool64/brick-starter-kit/internal/infra/service"
	"github.com/bool64/brick/runtime"
	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/require"
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
