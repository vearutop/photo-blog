package schema

import (
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest/openapi"
)

// SetupOpenapiCollector configures OpenAPI schema.
func SetupOpenapiCollector(c *openapi.Collector) {
	SetupJSONSchemaReflector(&c.Reflector().Reflector)

	c.Reflector().SpecEns().Info.Title = "Photo Blog"
}

func SetupJSONSchemaReflector(r *jsonschema.Reflector) {
	// No customizations yet.
}
