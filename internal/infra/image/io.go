package image

import (
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
)

func GrayJPEG(r io.Reader) (*image.Gray, error) {
	img, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray, nil
}

func Gray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray
}

func SaveJPEG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, nil)
}
