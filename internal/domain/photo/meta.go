package photo

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/image-prompt/multi"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
)

type Meta struct {
	uniq.Head

	Data sqluct.JSON[MetaData] `db:"data" json:"data"`
}

// Deprecated: use Label.
type ImageLabel struct {
	Model string  `json:"model"`
	Text  string  `json:"text"`
	Score float64 `json:"score,omitempty"`
}

type Label struct {
	Text  string      `json:"text,omitempty"`
	Score float64     `json:"score,omitempty"`
	Box   *PercentBox `json:"box,omitempty"`
}

type Face struct {
	Box        *PercentBox  `json:"box"`
	Descriptor []float64    `json:"descriptor"`
	Marks      []PercentBox `json:"marks"`
}

type PercentBox struct {
	Top    float32 `json:"t%"  description:"Top position in percent of image height."`
	Left   float32 `json:"l%" description:"Left position in percent of image width."`
	Height float32 `json:"h%,omitempty" description:"Height of the box in percent of image height."`
	Width  float32 `json:"w%,omitempty" description:"Width of the box in percent of image width."`
}

type MetaData struct {
	ImageClassification []ImageLabel        `json:"image_classification,omitempty"`
	Faces               *[]faces.GoFaceFace `json:"faces,omitempty"`
	GeoLabel            *string             `json:"geo_label,omitempty"`
	CFResnet50          *[]Label            `json:"cf_resnet_50,omitempty"`
	CFLlavaDescription  *string             `json:"cf_llava_description,omitempty"`
	CFDetrResnet        *[]Label            `json:"cf_detr_resnet,omitempty"`
	FaceVectors         *[]Face             `json:"face_vectors,omitempty"`
	ImageDescriptions   []multi.Result      `json:"image_descriptions,omitempty"`
}
