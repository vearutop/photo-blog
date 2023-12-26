package main

import (
	"encoding/json"
	"log"

	"github.com/swaggest/jsonschema-go"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

func main() {
	r := jsonschema.Reflector{}

	s, err := r.Reflect(jsonschema.OneOf(
		photo.Album{},
		photo.Image{},
		photo.AlbumSettings{},
	))
	if err != nil {
		log.Fatal(err)
	}

	j, _ := json.Marshal(s)

	println(string(j))
}
