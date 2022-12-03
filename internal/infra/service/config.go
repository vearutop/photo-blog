package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

// Name is the name of this application or service.
const Name = "photo-blog"

// Config defines application configuration.
type Config struct {
	brick.BaseConfig

	Database database.Config `split_words:"true"`
	Jaeger   jaeger.Config   `split_words:"true"`

	Thumb image.ThumbConfig `split_words:"true"`
}
