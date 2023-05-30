package photo

import (
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Gpx struct {
	uniq.Head

	Path string `db:"path" json:"path"`
}
