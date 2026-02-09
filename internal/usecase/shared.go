package usecase

import (
	"github.com/swaggest/rest/request"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type hashInPath struct {
	Hash uniq.Hash `path:"hash"`
	request.EmbeddedSetter
}
