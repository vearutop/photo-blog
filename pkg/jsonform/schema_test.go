package jsonform_test

import (
	"github.com/vearutop/photo-blog/pkg/jsonform"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
	"github.com/vearutop/photo-blog/internal/usecase/control"
)

func TestRepository_AddSchema(t *testing.T) {
	repo := jsonform.NewRepository(&jsonschema.Reflector{})

	assert.NoError(t, repo.AddSchema("test", control.UpdateImageInput{}))
	assertjson.EqualMarshal(t, []byte(`{
	  "form":[
		{"key":"hash","type":"hidden"},{"key":"exif.created_at","type":"hidden"},
		{"key":"exif.hash","type":"hidden"},{"key":"exif.rating"},
		{"key":"exif.exposure_time"},{"key":"exif.exposure_time_sec"},
		{"key":"exif.f_number"},{"key":"exif.focal_length"},
		{"key":"exif.iso_speed"},{"key":"exif.lens_model"},
		{"key":"exif.camera_make"},{"key":"exif.camera_model"},
		{"key":"exif.software"},{"key":"exif.digitized"},
		{"key":"exif.projection_type"},{"key":"gps.created_at","type":"hidden"},
		{"key":"gps.hash","type":"hidden"},{"key":"gps.altitude"},
		{"key":"gps.longitude"},{"key":"gps.latitude"},{"key":"gps.time"},
		{"key":"descriptions[].created_at","type":"hidden"},
		{"key":"descriptions[].hash","type":"hidden"},
		{"key":"descriptions[].locale"},
		{"key":"descriptions[].text","type":"textarea"},{"key":"descriptions"},
		{"type":"submit","title":"Submit"}
	  ],
	  "schema":{
		"properties":{
		  "descriptions":{
			"items":{
			  "properties":{
				"created_at":{
				  "title":"Created At","description":"Timestamp of creation.",
				  "type":"string","format":"date-time"
				},
				"hash":{
				  "title":"Hash Id","description":"Unique hash value.",
				  "type":"string"
				},
				"locale":{"title":"Locale","examples":["en-US"],"type":"string"},
				"text":{"type":"string"}
			  },
			  "type":"object"
			},
			"type":"array"
		  },
		  "exif":{
			"properties":{
			  "camera_make":{"title":"Manufacturer","type":"string"},
			  "camera_model":{"title":"Camera","type":"string"},
			  "created_at":{
				"title":"Created At","description":"Timestamp of creation.",
				"type":"string","format":"date-time"
			  },
			  "digitized":{"title":"Digitized","type":["null","string"],"format":"date-time"},
			  "exposure_time":{"title":"Exposure","type":"string"},
			  "exposure_time_sec":{"title":"Exposure (sec.)","type":"number"},
			  "f_number":{"title":"Aperture","type":"number"},
			  "focal_length":{"title":"Focal length","type":"number"},
			  "hash":{
				"title":"Hash Id","description":"Unique hash value.",
				"type":"string"
			  },
			  "iso_speed":{"title":"ISO","type":"integer"},
			  "lens_model":{"title":"Lens","type":"string"},
			  "projection_type":{"title":"Projection","type":"string"},
			  "rating":{"title":"Rating","maximum":5,"minimum":0,"type":"integer"},
			  "software":{"title":"Software","type":"string"}
			},
			"type":"object"
		  },
		  "gps":{
			"properties":{
			  "altitude":{"type":"number"},
			  "created_at":{
				"title":"Created At","description":"Timestamp of creation.",
				"type":"string","format":"date-time"
			  },
			  "hash":{
				"title":"Hash Id","description":"Unique hash value.",
				"type":"string"
			  },
			  "latitude":{"type":"number"},"longitude":{"type":"number"},
			  "time":{"type":"string","format":"date-time"}
			},
			"type":"object"
		  },
		  "hash":{"type":"string"}
		},
		"type":"object"
	  }
	}`), repo.Schema("test"))
}
