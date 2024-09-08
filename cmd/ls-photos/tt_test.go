package main_test

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/draw"
)

type spriter struct {
	baseName      string
	fileNumber    int
	currentHeight int
	width         int
	maxHeight     int
	pending       []string
	sprite        *image.RGBA
}

func (s *spriter) AddFile(fn string) (spritePath string, offset int, err error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", 0, err
	}

	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		return "", 0, fmt.Errorf("reading JPEG: %w", err)
	}

	offset = s.currentHeight

	if s.currentHeight+cfg.Height > s.maxHeight {
		offset = 0

		if err := s.Flush(); err != nil {
			return "", 0, err
		}
	}

	s.currentHeight += cfg.Height
	s.pending = append(s.pending, fn)

	return s.filename(), offset, nil
}

func (s *spriter) filename() string {
	return fmt.Sprintf("%s_%d.jpg", s.baseName, s.fileNumber)
}

func (s *spriter) processFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return err
	}

	// println("size", img.Bounds().Size().X, img.Bounds().Size().Y, img.Bounds().Dy())

	height := img.Bounds().Size().Y

	draw.Draw(s.sprite, image.Rectangle{
		Min: image.Point{Y: s.currentHeight},
		Max: image.Point{X: s.width, Y: s.currentHeight + height},
	}, img, image.Point{}, draw.Src)

	s.currentHeight += height

	return nil
}

func (s *spriter) Flush() error {
	if len(s.pending) == 0 {
		return nil
	}

	r := image.Rectangle{Max: image.Point{X: s.width, Y: s.currentHeight}}
	s.sprite = image.NewRGBA(r)
	s.currentHeight = 0

	for _, fn := range s.pending {
		if err := s.processFile(fn); err != nil {
			return err
		}
	}

	sf, err := os.Create(s.filename())
	if err != nil {
		return err
	}
	defer sf.Close()

	if err := jpeg.Encode(sf, s.sprite, &jpeg.Options{Quality: 90}); err != nil {
		return err
	}

	println("flushed ", s.filename())

	s.fileNumber++
	s.pending = nil
	s.currentHeight = 0
	s.sprite = nil

	return nil
}

func TestSprite(t *testing.T) {
	files, err := os.ReadDir("testdata")
	require.NoError(t, err)

	s := spriter{
		baseName:   "sprite",
		fileNumber: 0,
		width:      600,
		maxHeight:  20000,
	}

	for _, fn := range files {
		println(fn.Name())

		sfn, offset, err := s.AddFile("testdata/" + fn.Name())
		require.NoError(t, err)
		println(fn.Name(), sfn, offset)
	}

	require.NoError(t, s.Flush())
}
