package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
)

// Name is the name of this application or service.
const Name = "brick-starter-kit"

// Config defines application configuration.
type Config struct {
	brick.BaseConfig

	Database database.Config `split_words:"true"`
	Jaeger   jaeger.Config   `split_words:"true"`
}
