package short_test

import (
	"testing"

	"github.com/vearutop/photo-blog/cmd/shortener/internal/domain/short"
)

func BenchmarkMakeToken(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t := short.MakeToken("fooooooooooooooo", i, 7)
		if len(t) != 7 {
			b.Fail()
		}
	}
}
