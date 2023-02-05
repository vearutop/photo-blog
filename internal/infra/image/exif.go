package image

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type Meta struct {
	exif map[string]any
	photo.Exif
	GpsInfo *photo.Gps `json:"gps_info,omitempty"`
}

func (m Meta) ExifData() map[string]any {
	return m.exif
}

func ReadMeta(r io.ReadSeeker) (Meta, error) {
	res := Meta{}

	//    <rdf:RDF>
	//      <rdf:Description rdf:about="">
	//        <ns2:ProjectionType>equirectangular</ns2:ProjectionType>
	//        <ns2:UsePanoramaViewer>True</ns2:UsePanoramaViewer>
	//        <ns2:CroppedAreaLeftPixels>0</ns2:CroppedAreaLeftPixels>
	//        <ns2:CroppedAreaTopPixels>0</ns2:CroppedAreaTopPixels>
	//        <ns2:PoseHeadingDegrees>0.0</ns2:PoseHeadingDegrees>
	//        <ns2:PosePitchDegrees>2.5</ns2:PosePitchDegrees>
	//        <ns2:PoseRollDegrees>-0.4</ns2:PoseRollDegrees>
	//<------><ns2:StitchingSoftware> Samsung Gear 360 Mac </ns2:StitchingSoftware>
	//      </rdf:Description>
	//    </rdf:RDF>
	//  </ns0:xmpmeta>.

	xmp := Span{
		Start: []byte(":xmpmeta"),
		End:   []byte(":xmpmeta>"),
	}
	if err := FindSpans(r, &xmp); err != nil {
		return res, fmt.Errorf("find xmp: %w", err)
	}

	if xmp.Found {
		//<ns2:ProjectionType>equirectangular</ns2:ProjectionType>
		pt := Span{Start: []byte("<ns2:ProjectionType>"), End: []byte("</ns2:ProjectionType>"), Trim: true}
		r1 := Span{Start: []byte(`xmp:Rating="`), End: []byte(`"`), Trim: true}
		r2 := Span{Start: []byte(`<xmp:Rating>`), End: []byte(`</xmp:Rating>`), Trim: true}

		_ = FindSpans(bytes.NewReader(xmp.Result), &pt, &r1, &r2)
		if pt.Found {
			res.ProjectionType = string(pt.Result)
		}

		rs := ""
		if r1.Found {
			rs = string(r1.Result)
		} else if r2.Found {
			rs = string(r2.Result)
		}

		if rs != "" {
			rating, err := strconv.Atoi(rs)
			if err != nil {
				return res, fmt.Errorf("parse xmp rating %q: %w", rs, err)
			}

			res.Rating = rating
		}
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return res, fmt.Errorf("seek: %w", err)
	}

	rawExif, err := exif.SearchAndExtractExifWithReader(r)
	if err != nil {
		return res, fmt.Errorf("search exif: %w", err)
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return res, fmt.Errorf("ifd mapping: %w", err)
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return res, fmt.Errorf("exif collect: %w", err)
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
		return res, fmt.Errorf("enumerate tags: %w", err)
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

type Span struct {
	Before        int
	Start, End    []byte
	CaseSensitive bool
	Result        []byte
	Found         bool
	Trim          bool
}

func FindSpans(r io.Reader, spans ...*Span) error {
	l := 4096

	for _, s := range spans {
		if len(s.Start) > l {
			l = len(s.Start)
		}

		if len(s.End) > l {
			l = len(s.End)
		}
	}

	dbl := make([]byte, 2*l)
	buf := make([]byte, l)
	pending := len(spans)

	for {
		if pending == 0 {
			return nil
		}

		_, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		copy(dbl[l:], buf)

		for _, s := range spans {
			if s.Found {
				continue
			}

			started := len(s.Result) > 0
			chunk := dbl
			if started {
				chunk = buf
			}

			if !started {
				if i := bytes.Index(chunk, s.Start); i != -1 {
					if s.Trim {
						i += len(s.Start)
					}

					i -= s.Before
					if i < 0 {
						i = 0
					}
					chunk = chunk[i:]
					started = true
				}
			}

			if started {
				s.Result = append(s.Result, chunk...)

				if i := bytes.Index(s.Result, s.End); i != -1 {
					if s.Trim {
						s.Result = s.Result[:i]
					} else {
						s.Result = s.Result[:i+len(s.End)]
					}
					s.Found = true
					pending--
				}
			}
		}

		copy(dbl, buf)
	}
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
