package uniq_test

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

func TestHash_MarshalJSON(t *testing.T) {
	for _, i := range []int64{
		math.MaxInt64,
		math.MinInt64,
		0,
		100000000000000,
		-100000000000000,
		1281587067978813000,
		-1281587067978813000,
	} {
		t.Run(strconv.Itoa(int(i)), func(t *testing.T) {
			h := uniq.Hash(i)

			s := h.String()
			j, err := json.Marshal(s)
			assert.NoError(t, err)

			tt, err := h.MarshalText()
			assert.NoError(t, err)

			h2 := uniq.Hash(0)
			assert.NoError(t, h2.UnmarshalText([]byte(s)))
			assert.Equal(t, h, h2)

			h2 = 0
			assert.NoError(t, h2.UnmarshalText(tt))
			assert.Equal(t, h, h2)

			h2 = 0
			assert.NoError(t, json.Unmarshal(j, &h2))
			assert.Equal(t, h, h2)
		})
	}
}
