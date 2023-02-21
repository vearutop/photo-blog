package schema

import (
	"fmt"
	"github.com/swaggest/jsonschema-go"
)

func SetupJSONSchemaReflector(r *jsonschema.Reflector) {
	// No customizations yet.
}

type Repository struct {
	reflector *jsonschema.Reflector
	schemas   map[string]jsonschema.Schema
}

func NewRepository(reflector *jsonschema.Reflector) *Repository {
	r := Repository{}
	r.reflector = reflector
	r.schemas = make(map[string]jsonschema.Schema)

	return &r
}

func (r *Repository) AddSchema(name string, value any) error {
	if r.schemas == nil {
		r.schemas = make(map[string]jsonschema.Schema)
	}

	schema, err := r.reflector.Reflect(value, jsonschema.InlineRefs)
	if err != nil {
		return fmt.Errorf("reflecting %s schema: %w", name, err)
	}

	r.schemas[name] = schema

	return nil
}

func (r *Repository) Schema(name string) jsonschema.Schema {
	return r.schemas[name]
}
