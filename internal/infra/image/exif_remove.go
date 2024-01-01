package image

import (
	"bytes"
	"errors"
	"image/jpeg"

	"github.com/dsoprea/go-exif/v3"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure"
)

func Remove(data []byte) ([]byte, error) {
	type MediaContext struct {
		MediaType string
		RootIfd   *exif.Ifd
		RawExif   []byte
		Media     interface{}
	}

	jmp := jpegstructure.NewJpegMediaParser()
	filtered := []byte{}

	if jmp.LooksLikeFormat(data) {
		sl, err := jmp.ParseBytes(data)
		if err != nil {
			return nil, err
		}

		_, rawExif, err := sl.Exif()
		if err != nil {
			return data, nil
		}

		startExifBytes := 0
		endExifBytes := 0

		if bytes.Contains(data, rawExif) {
			for i := 0; i < len(data)-len(rawExif); i++ {
				if bytes.Compare(data[i:i+len(rawExif)], rawExif) == 0 {
					startExifBytes = i
					endExifBytes = i + len(rawExif)
					break
				}
			}
			fill := make([]byte, len(data[startExifBytes:endExifBytes]))
			copy(data[startExifBytes:endExifBytes], fill)
		}

		filtered = data

		_, err = jpeg.Decode(bytes.NewReader(filtered))
		if err != nil {
			return nil, errors.New("EXIF removal corrupted " + err.Error())
		}

	}

	return filtered, nil
}
