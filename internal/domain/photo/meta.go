package photo

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Meta struct {
	uniq.Head

	Data sqluct.JSON[MetaData] `db:"data" json:"data"`
}

type ImageLabel struct {
	Model string  `json:"model"`
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

type MetaData struct {
	ImageClassification []ImageLabel `json:"image_classification"`
}
