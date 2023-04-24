package image_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

func TestReadMeta(t *testing.T) {
	f, err := os.Open("_testdata/IMG_5612.jpg")
	defer func() {
		require.NoError(t, f.Close())
	}()
	require.NoError(t, err)

	m, err := image.ReadMeta(f)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "created_at":"0001-01-01T00:00:00Z","hash":"0","rating":5,
	  "exposure_time":"1/320","exposure_time_sec":0.003125,"f_number":11,
	  "focal_length":50,"iso_speed":100,"lens_model":"EF50mm f/1.8 STM",
	  "camera_make":"Canon","camera_model":"Canon EOS 6D",
	  "software":"Adobe Photoshop Camera Raw 14.3 (Windows)",
	  "digitized":"2022-12-08T13:24:48+01:00","projection_type":"",
	  "gps_info":{
		"created_at":"0001-01-01T00:00:00Z","hash":"0","altitude":2092.4,
		"longitude":-16.678148333333333,"latitude":28.21437,
		"time":"2022-12-08T11:24:44Z"
	  }
	}`), m)

	assertjson.EqualMarshal(t, []byte(`{
	  "IFD/DateTime":"2023:01:01 23:12:20",
	  "IFD/Exif/ApertureValue":"6918863/1000000",
	  "IFD/Exif/BodySerialNumber":"203020001007","IFD/Exif/ColorSpace":"[1]",
	  "IFD/Exif/CustomRendered":"[0]",
	  "IFD/Exif/DateTimeDigitized":"2022:12:08 13:24:48",
	  "IFD/Exif/DateTimeOriginal":"2022:12:08 13:24:48",
	  "IFD/Exif/ExifVersion":"0231","IFD/Exif/ExposureBiasValue":"[0/1]",
	  "IFD/Exif/ExposureMode":"[1]","IFD/Exif/ExposureProgram":"[1]",
	  "IFD/Exif/ExposureTime":"1/320","IFD/Exif/FNumber":11,"IFD/Exif/Flash":"[16]",
	  "IFD/Exif/FocalLength":50,"IFD/Exif/FocalPlaneResolutionUnit":"[3]",
	  "IFD/Exif/FocalPlaneXResolution":"49807360/32768",
	  "IFD/Exif/FocalPlaneYResolution":"49807360/32768",
	  "IFD/Exif/ISOSpeedRatings":"[100]","IFD/Exif/LensModel":"EF50mm f/1.8 STM",
	  "IFD/Exif/LensSerialNumber":"0000231f6a",
	  "IFD/Exif/LensSpecification":"[50/1 50/1 0/0 0/0]",
	  "IFD/Exif/MaxApertureValue":"175/100","IFD/Exif/MeteringMode":"[5]",
	  "IFD/Exif/OffsetTime":"+01:00","IFD/Exif/RecommendedExposureIndex":"[100]",
	  "IFD/Exif/SceneCaptureType":"[0]","IFD/Exif/SensitivityType":"[2]",
	  "IFD/Exif/ShutterSpeedValue":"[8321928/1000000]",
	  "IFD/Exif/SubSecTimeDigitized":"88","IFD/Exif/SubSecTimeOriginal":"88",
	  "IFD/Exif/WhiteBalance":"[0]","IFD/GPSInfo/GPSAltitude":"2092.4",
	  "IFD/GPSInfo/GPSAltitudeRef":"00","IFD/GPSInfo/GPSDOP":"4.1",
	  "IFD/GPSInfo/GPSDateStamp":"2022:12:08",
	  "IFD/GPSInfo/GPSLatitude":"[28/1 128622000/10000000 0/1]",
	  "IFD/GPSInfo/GPSLatitudeRef":"N",
	  "IFD/GPSInfo/GPSLongitude":"[16/1 406889000/10000000 0/1]",
	  "IFD/GPSInfo/GPSLongitudeRef":"W","IFD/GPSInfo/GPSMapDatum":"WGS-84",
	  "IFD/GPSInfo/GPSMeasureMode":"3","IFD/GPSInfo/GPSSatellites":"12",
	  "IFD/GPSInfo/GPSStatus":"A",
	  "IFD/GPSInfo/GPSTimeStamp":"[11/1 24/1 44003/1000]",
	  "IFD/GPSInfo/GPSVersionID":"02 03 00 00","IFD/Make":"Canon",
	  "IFD/Model":"Canon EOS 6D","IFD/ResolutionUnit":"[2]",
	  "IFD/Software":"Adobe Photoshop Camera Raw 14.3 (Windows)",
	  "IFD/XResolution":300,"IFD/YResolution":300
	}`), m.ExifData())
}

func TestReadMeta_360(t *testing.T) {
	f, err := os.Open("_testdata/360.jpg")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	m, err := image.ReadMeta(f)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "created_at":"0001-01-01T00:00:00Z","hash":"0","rating":0,
	  "exposure_time":"1/1250","exposure_time_sec":0.0008,"f_number":2,
	  "focal_length":1.2,"iso_speed":100,"lens_model":"","camera_make":"SAMSUNG",
	  "camera_model":"SM-C200","software":"Samsung Gear 360 Mac",
	  "digitized":"2022-08-02T10:52:26Z","projection_type":"equirectangular"
	}`), m)

	assertjson.EqualMarshal(t, []byte(`{
	  "IFD/DateTime":"2022:07:27 14:02:23","IFD/Exif/ColorSpace":"[1]",
	  "IFD/Exif/Contrast":"[0]","IFD/Exif/DateTimeDigitized":"2022:08:02 10:52:26",
	  "IFD/Exif/DateTimeOriginal":"2022:08:02 10:52:26",
	  "IFD/Exif/DigitalZoomRatio":"1","IFD/Exif/ExifVersion":"0230",
	  "IFD/Exif/ExposureBiasValue":"[6/10]","IFD/Exif/ExposureMode":"[0]",
	  "IFD/Exif/ExposureProgram":"[2]","IFD/Exif/ExposureTime":"1/1250",
	  "IFD/Exif/FNumber":"2","IFD/Exif/Flash":"[0]","IFD/Exif/FocalLength":"1.2",
	  "IFD/Exif/FocalLengthIn35mmFilm":"[6]","IFD/Exif/ISOSpeedRatings":"[100]",
	  "IFD/Exif/LightSource":"[0]","IFD/Exif/MeteringMode":"[5]",
	  "IFD/Exif/PixelXDimension":"[7776]","IFD/Exif/PixelYDimension":"[3888]",
	  "IFD/Exif/Saturation":"[0]","IFD/Exif/SceneCaptureType":"[0]",
	  "IFD/Exif/SensingMethod":"[2]","IFD/Exif/Sharpness":"[0]",
	  "IFD/Exif/WhiteBalance":"[0]","IFD/ImageDescription":"SAMSUNG    ",
	  "IFD/Make":"SAMSUNG","IFD/Model":"SM-C200","IFD/ResolutionUnit":"[2]",
	  "IFD/Software":"Samsung Gear 360 Mac","IFD/XResolution":350,
	  "IFD/YResolution":350
	}`), m.ExifData())
}

func TestFindSpans(t *testing.T) {
	f, err := os.Open("_testdata/IMG_5612.jpg")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	xmp := image.Span{
		Before: 10,
		Start:  []byte(`<x:xmpmeta`),
		End:    []byte(`</x:xmpmeta>`),
	}

	assert.NoError(t, image.FindSpans(f, &xmp))

	println(string(xmp.Result))
}

func TestFindSpans_360(t *testing.T) {
	f, err := os.Open("_testdata/360.jpg")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	xmp := image.Span{
		Start: []byte(`<x:xmpmeta`),
		End:   []byte(`</x:xmpmeta>`),
	}

	ns0xmp := image.Span{
		Start: []byte(`<ns0:xmpmeta`),
		End:   []byte(`</ns0:xmpmeta>`),
	}

	assert.NoError(t, image.FindSpans(f, &ns0xmp, &xmp))

	assert.Equal(t, `<ns0:xmpmeta xmlns:ns0="adobe:ns:meta/" xmlns:ns2="http://ns.google.com/photos/1.0/panorama/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmptk="SAMSUNG 360CAM">
    <rdf:RDF>
      <rdf:Description rdf:about="">
        <ns2:ProjectionType>equirectangular</ns2:ProjectionType>
        <ns2:UsePanoramaViewer>True</ns2:UsePanoramaViewer>
        <ns2:CroppedAreaLeftPixels>0</ns2:CroppedAreaLeftPixels>
        <ns2:CroppedAreaTopPixels>0</ns2:CroppedAreaTopPixels>
        <ns2:PoseHeadingDegrees>0.0</ns2:PoseHeadingDegrees>
        <ns2:PosePitchDegrees>2.5</ns2:PosePitchDegrees>
        <ns2:PoseRollDegrees>-0.4</ns2:PoseRollDegrees>
	<ns2:StitchingSoftware> Samsung Gear 360 Mac </ns2:StitchingSoftware>
      </rdf:Description>
    </rdf:RDF>
  </ns0:xmpmeta>`, string(ns0xmp.Result))

	assert.False(t, xmp.Found)
}
