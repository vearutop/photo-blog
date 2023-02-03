package image_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

func TestReadMeta(t *testing.T) {
	f, err := os.Open("_testdata/IMG_5612.jpg")
	require.NoError(t, err)

	m, err := image.ReadMeta(f)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "Rating":5,"ExposureTime":"1/320","ExposureTimeSec":0.003125,
	  "FNumber":11,"FocalLength":50,"ISOSpeedRatings":100,
	  "LensModel":"EF50mm f/1.8 STM","CameraMake":"Canon",
	  "CameraModel":"Canon EOS 6D",
	  "Software":"Adobe Photoshop Camera Raw 14.3 (Windows)",
      "Digitized": "2022-12-08T13:24:48+01:00",
	  "GpsInfo":{
		"Altitude":2092.4,"Longitude":-16.678148333333333,"Latitude":28.21437,
		"Time":"2022-12-08T11:24:44Z"
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

	m, err := image.ReadMeta(f)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "Rating":5,"ExposureTime":"1/320","ExposureTimeSec":0.003125,
	  "FNumber":11,"FocalLength":50,"ISOSpeedRatings":100,
	  "LensModel":"EF50mm f/1.8 STM","CameraMake":"Canon",
	  "CameraModel":"Canon EOS 6D",
	  "Software":"Adobe Photoshop Camera Raw 14.3 (Windows)",
      "Digitized": "2022-12-08T13:24:48+01:00",
	  "GpsInfo":{
		"Altitude":2092.4,"Longitude":-16.678148333333333,"Latitude":28.21437,
		"Time":"2022-12-08T11:24:44Z"
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
