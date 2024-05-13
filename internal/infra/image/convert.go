package image

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/webp"
)

// ToJpeg converts a PNG/GIF/WEBP image to JPEG format.
func ToJpeg(imageBytes []byte) ([]byte, error) {
	// DetectContentType detects the content type
	contentType := http.DetectContentType(imageBytes)

	var (
		img image.Image
		err error
	)

	switch contentType {
	case "image/jpeg":
		return imageBytes, nil
	case "image/gif":
		img, err = gif.Decode(bytes.NewReader(imageBytes))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imageBytes))
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(imageBytes))
	default:
		return nil, fmt.Errorf("unsupported image type: %s", contentType)
	}

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	// encode the image as a JPEG file
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
