package sprite

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
	"strconv"

	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type Vertical struct {
	BaseName      string
	Width         int
	MaxHeight     int
	fileNumber    int
	currentHeight int
	pending       []photo.Thumb
	sprite        *image.RGBA
}

func NewVertical(width int) *Vertical {
	return &Vertical{
		BaseName:  "sprite_w" + strconv.Itoa(width),
		Width:     width,
		MaxHeight: 20000,
	}
}

func (s *Vertical) AddFile(fn string) (spritePath string, offset int, err error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", 0, err
	}

	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		return "", 0, fmt.Errorf("reading JPEG: %w", err)
	}

	offset = s.currentHeight

	if s.currentHeight+cfg.Height > s.MaxHeight {
		offset = 0

		if err := s.Flush(); err != nil {
			return "", 0, err
		}
	}

	th := photo.Thumb{
		FilePath: fn,
		Width:    uint(cfg.Width),
		Height:   uint(cfg.Height),
	}

	s.currentHeight += cfg.Height
	s.pending = append(s.pending, th)

	return s.filename(), offset, nil
}

func (s *Vertical) AddThumb(th photo.Thumb) (spritePath string, offset int, err error) {
	if s.currentHeight+int(th.Height) > s.MaxHeight {
		offset = 0

		if err := s.Flush(); err != nil {
			return "", 0, err
		}
	}

	offset = s.currentHeight

	s.currentHeight += int(th.Height)
	s.pending = append(s.pending, th)

	return s.filename(), offset, nil
}

func (s *Vertical) filename() string {
	return fmt.Sprintf("%s_%d.jpg", s.BaseName, s.fileNumber)
}

func (s *Vertical) processFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return s.processBytes(f)
}

func (s *Vertical) processBytes(data io.Reader) error {
	img, err := jpeg.Decode(data)
	if err != nil {
		return err
	}

	// println("size", img.Bounds().Size().X, img.Bounds().Size().Y, img.Bounds().Dy())

	height := img.Bounds().Size().Y

	draw.Draw(s.sprite, image.Rectangle{
		Min: image.Point{Y: s.currentHeight},
		Max: image.Point{X: s.Width, Y: s.currentHeight + height},
	}, img, image.Point{}, draw.Src)

	s.currentHeight += height

	return nil
}

func (s *Vertical) Flush() error {
	if len(s.pending) == 0 {
		return nil
	}

	r := image.Rectangle{Max: image.Point{X: s.Width, Y: s.currentHeight}}
	s.sprite = image.NewRGBA(r)
	s.currentHeight = 0

	for _, th := range s.pending {
		if th.FilePath != "" {
			if err := s.processFile(th.FilePath); err != nil {
				return err
			}
		} else if len(th.Data) > 0 {
			if err := s.processBytes(bytes.NewReader(th.Data)); err != nil {
				return err
			}
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
