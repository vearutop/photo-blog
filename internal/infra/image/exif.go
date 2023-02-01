package image

import (
	"bytes"
	"fmt"
	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"io"
	"strconv"
	"time"
)

type GpsInfo struct {
	Altitude  int
	Longitude float64
	Latitude  float64
	Time      time.Time
}

type Meta struct {
	Rating  int
	Exif    map[string]string
	GpsInfo *GpsInfo
}

func ReadMeta(r io.ReadSeeker) (Meta, error) {
	res := Meta{}

	const ratingPref = `xmp:Rating="`
	cc, err := find(r, []byte(ratingPref), len(ratingPref)+2)
	if err != nil {
		return res, err
	}

	if len(cc) >= len(ratingPref)+2 {
		rating, err := strconv.Atoi(string(cc)[len(ratingPref) : len(ratingPref)+1])
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
	if err != nil {
		return res, err
	}

	gi, err := ifd.GpsInfo()
	if err != nil {
		return res, err
	}

	if gi != nil {
		g := GpsInfo{}
		g.Altitude = gi.Altitude
		g.Longitude = gi.Longitude.Decimal()
		g.Latitude = gi.Latitude.Decimal()
		g.Time = gi.Timestamp

		res.GpsInfo = &g
	}

	res.Exif = make(map[string]string)
	cb := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		v, _ := ite.Value()
		var s string

		if r, ok := v.([]exifcommon.Rational); ok && len(r) == 1 {
			f := float64(r[0].Numerator) / float64(r[0].Denominator)
			s = fmt.Sprintf("%.1f", f)
		} else {
			s, err = ite.Format()
			if err != nil {
				s = err.Error()
			}
		}

		res.Exif[ite.IfdPath()+"/"+ite.TagName()] = s

		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(cb)
	if err != nil {
		return res, err
	}

	return res, nil
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
