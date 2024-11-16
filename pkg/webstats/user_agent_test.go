package webstats_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vearutop/photo-blog/pkg/webstats"
)

func TestIsBot(t *testing.T) {
	assert.False(t, webstats.IsBot("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Safari/605.1.15"))
}
