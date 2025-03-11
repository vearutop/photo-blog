package image

import (
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
)

func ToRGBA(src image.Image) *image.RGBA {
	if dst, ok := src.(*image.RGBA); ok {
		return dst
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

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

func LoadJPEG(imgSrc string) (image.Image, error) {
	if strings.HasPrefix(imgSrc, "http://") || strings.HasPrefix(imgSrc, "https://") {
		resp, err := http.Get(imgSrc)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		return jpeg.Decode(resp.Body)
	}

	f, err := os.Open(imgSrc)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return jpeg.Decode(f)
}
