package image

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type Meta struct {
	exif map[string]any
	photo.Exif
	GpsInfo *photo.Gps
}

func (m Meta) ExifData() map[string]any {
	return m.exif
}

func ReadMeta(r io.ReadSeeker) (Meta, error) {
	res := Meta{}

	// TODO properly scan and process XMP tags, including bespoke xmp:Label from Photoshop and ACDSee.
	const ratingPref = `xmp:Rating` // Can be `<xmp:Rating>5</xmp:Rating>` or `xmp:Rating="5"`.
	cc, err := find(r, []byte(ratingPref), len(ratingPref)+3)
	if err != nil {
		return res, err
	}

	if len(cc) >= len(ratingPref)+3 {
		rating, err := strconv.Atoi(strings.Trim(string(cc)[len(ratingPref)+1:len(ratingPref)+3], `"></`))
		if err != nil {
			return res, err
		}

		res.Rating = rating
	}

	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return res, err
	}

	rawExif, err := exif.SearchAndExtractExifWithReader(r)
	if err != nil {
		return res, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return res, err
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return res, err
	}

	ifd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity)
	if err == nil {
		if gi, err := ifd.GpsInfo(); err == nil && gi != nil {
			g := photo.Gps{}
			g.Altitude = float64(gi.Altitude)
			g.Longitude = gi.Longitude.Decimal()
			g.Latitude = gi.Latitude.Decimal()
			g.GpsTime = gi.Timestamp

			res.GpsInfo = &g
		}
	}

	digTime := ""
	digTimeOffset := ""

	res.exif = make(map[string]any)
	cb := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		key := ite.IfdPath() + "/" + ite.TagName()
		v, _ := ite.Value()

		switch key {
		case "IFD/Exif/ExposureTime":
			res.ExposureTimeSec = extractExifFloat(v)
			res.ExposureTime = extractExifStr(v)
		case "IFD/Exif/FNumber":
			res.FNumber = extractExifFloat(v)
		case "IFD/Exif/FocalLength":
			res.FocalLength = extractExifFloat(v)
		case "IFD/Exif/ISOSpeedRatings":
			res.ISOSpeed = extractExifInt(v)
		case "IFD/Exif/LensModel":
			res.LensModel, _ = ite.Format()
		case "IFD/Model":
			res.CameraModel, _ = ite.Format()
		case "IFD/Make":
			res.CameraMake, _ = ite.Format()
		case "IFD/Software":
			res.Software, _ = ite.Format()
		case "IFD/GPSInfo/GPSAltitude":
			if res.GpsInfo != nil {
				res.GpsInfo.Altitude = extractExifFloat(v)
			}
		case "IFD/Exif/DateTimeDigitized":
			digTime, _ = ite.Format()
		case "IFD/Exif/OffsetTime":
			digTimeOffset, _ = ite.Format()
		}

		if digTime != "" {
			// "2022:12:08 13:24:48"
			// "+01:00"
			if digTimeOffset != "" {
				t, err := time.Parse("2006:01:02 15:04:05-07:00", digTime+digTimeOffset)
				if err == nil {
					res.Digitized = &t
				}
			} else {
				t, err := time.Parse("2006:01:02 15:04:05", digTime)
				if err == nil {
					res.Digitized = &t
				}
			}
		}

		if vv := extractExifValue(v); vv != nil {
			res.exif[key] = vv
		} else {
			s, err := ite.Format()
			if err != nil {
				s = err.Error()
			}
			res.exif[key] = s
		}

		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(cb)
	if err != nil {
		return res, err
	}

	return res, nil
}

func extractExifInt(v any) int {
	switch vv := v.(type) {
	case []exifcommon.Rational:
		if len(vv) == 1 {
			i := vv[0]
			if i.Denominator == 1 {
				return int(i.Numerator)
			}

			return int(i.Numerator / i.Denominator)
		}
	case []uint16:
		if len(vv) == 1 {
			return int(vv[0])
		}
	}

	return 0
}

func extractExifFloat(v any) float64 {
	switch vv := v.(type) {
	case []exifcommon.Rational:
		if len(vv) == 1 {
			i := vv[0]
			if i.Denominator == 1 {
				return float64(i.Numerator)
			}

			return float64(i.Numerator) / float64(i.Denominator)
		}
	}

	return 0
}

func extractExifStr(v any) string {
	switch vv := v.(type) {
	case []exifcommon.Rational:
		if len(vv) == 1 {
			i := vv[0]
			if i.Denominator == 1 {
				return strconv.Itoa(int(i.Numerator))
			}

			return fmt.Sprintf("%d/%d", i.Numerator, i.Denominator)
		}
	}

	return ""
}

func extractExifValue(v any) any {
	switch vv := v.(type) {
	case []exifcommon.Rational:
		if len(vv) == 1 {
			i := vv[0]
			if i.Denominator == 1 {
				return int(i.Numerator)
			}

			if vv[0].Denominator == 10 {
				return strconv.FormatFloat(float64(i.Numerator)/float64(i.Denominator), 'f', -1, 64)
			}

			return fmt.Sprintf("%d/%d", i.Numerator, i.Denominator)
		}
	}

	return nil
}

func find(r io.Reader, search []byte, resLen int) ([]byte, error) {
	l := 4096

	if l < len(search)+resLen {
		l = len(search) + resLen
	}

	dbl := make([]byte, 2*l)
	buf := make([]byte, l)

	for {
		_, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}

		copy(dbl[l:], buf)

		if i := bytes.Index(dbl, search); i != -1 {
			return dbl[i:], nil
		}

		copy(dbl, buf)
	}
}
