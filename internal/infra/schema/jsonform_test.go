package schema_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/usecase/control"
	"testing"
)

func TestRepository_AddSchema(t *testing.T) {
	repo := schema.NewRepository(&jsonschema.Reflector{})

	assert.NoError(t, repo.AddSchema("test", control.UpdateImageInput{}))
	assertjson.EqualMarshal(t, []byte(`{
	  "form":[
		{"key":"hash","type":"hidden"},{"key":"created_at","type":"hidden"},
		{"key":"hash","type":"hidden"},{"key":"rating"},{"key":"exposure_time"},
		{"key":"exposure_time_sec"},{"key":"f_number"},{"key":"focal_length"},
		{"key":"iso_speed"},{"key":"lens_model"},{"key":"camera_make"},
		{"key":"camera_model"},{"key":"software"},{"key":"digitized"},
		{"key":"projection_type"},{"key":"exif"},
		{"key":"created_at","type":"hidden"},{"key":"hash","type":"hidden"},
		{"key":"altitude"},{"key":"longitude"},{"key":"latitude"},{"key":"time"},
		{"key":"gps"},{"key":"created_at","type":"hidden"},
		{"key":"hash","type":"hidden"},{"key":"locale"},
		{"key":"text","type":"textarea"},{"key":"descriptions"},
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
			  "camera_make":{"type":"string"},"camera_model":{"type":"string"},
			  "created_at":{
				"title":"Created At","description":"Timestamp of creation.",
				"type":"string","format":"date-time"
			  },
			  "digitized":{"type":["null","string"],"format":"date-time"},
			  "exposure_time":{"type":"string"},
			  "exposure_time_sec":{"type":"number"},"f_number":{"type":"number"},
			  "focal_length":{"type":"number"},
			  "hash":{
				"title":"Hash Id","description":"Unique hash value.",
				"type":"string"
			  },
			  "iso_speed":{"type":"integer"},"lens_model":{"type":"string"},
			  "projection_type":{"type":"string"},"rating":{"type":"integer"},
			  "software":{"type":"string"}
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
