package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/brick/jaeger"
)

// Name is the name of this application or service.
const Name = "photo-blog"

// Config defines application configuration.
type Config struct {
	brick.BaseConfig

	StoragePath string        `split_words:"true" default:"./photo-blog-data/"`
	Jaeger      jaeger.Config `split_words:"true"`
}
