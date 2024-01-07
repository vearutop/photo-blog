package storage

import "github.com/bool64/sqluct"

type Stats struct {
	main   *sqluct.Storage
	thumbs *sqluct.Storage
}

func NewStats(main *sqluct.Storage, thumbs *sqluct.Storage) *Stats {
	return &Stats{
		main:   main,
		thumbs: thumbs,
	}
}
