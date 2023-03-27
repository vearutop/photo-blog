package schema

import (
	"fmt"
	"strings"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/refl"
)

func SetupJSONSchemaReflector(r *jsonschema.Reflector) {
	// No customizations yet.
}

type FormItem struct {
	Key       string `json:"key,omitempty" example:"longmood"`
	FormType  string `json:"type,omitempty" examples:"[\"textarea\",\"password\",\"wysihtml5\",\"submit\",\"color\",\"checkboxes\",\"radios\",\"fieldset\", \"help\", \"hidden\"]"`
	FormTitle string `json:"title,omitempty" example:"Submit"`

	ReadOnly bool `json:"readonly,omitempty"`

	Prepend        string            `json:"prepend,omitempty" example:"I feel"`
	Append         string            `json:"append,omitempty" example:"today"`
	NoTitle        bool              `json:"notitle,omitempty"`
	HtmlClass      string            `json:"htmlClass,omitempty" example:"usermood"`
	HtmlMetaData   map[string]string `json:"htmlMetaData,omitempty" example:"{\"style\":\"border: 1px solid blue\",\"data-title\":\"Mood\"}"`
	FieldHTMLClass string            `json:"fieldHtmlClass,omitempty" example:"input-xxlarge"`
	Placeholder    string            `json:"placeholder,omitempty" example:"incredibly and admirably great"`
	InlineTitle    string            `json:"inlinetitle,omitempty" example:"Check this box if you are over 18"`
	TitleMap       map[string]string `json:"titleMap,omitempty" description:"Title mapping for enum."`
	ActiveClass    string            `json:"activeClass,omitempty" example:"btn-success" description:"Button mode for radio buttons."`
	HelpValue      string            `json:"helpvalue,omitempty" example:"<strong>Click me!</strong>"`
}
type FormSchema struct {
	Form   []FormItem        `json:"form,omitempty"`
	Schema jsonschema.Schema `json:"schema"`
}

type Repository struct {
	reflector *jsonschema.Reflector
	schemas   map[string]FormSchema
}

func NewRepository(reflector *jsonschema.Reflector) *Repository {
	r := Repository{}
	r.reflector = reflector
	r.schemas = make(map[string]FormSchema)

	return &r
}

func (r *Repository) AddSchema(name string, value any) error {
	fs := FormSchema{}

	schema, err := r.reflector.Reflect(value, jsonschema.InlineRefs, jsonschema.InterceptProp(
		func(params jsonschema.InterceptPropParams) error {
			if params.PropertySchema.HasType(jsonschema.Object) {
				return nil
			}

			fi := FormItem{
				Key: strings.Join(append(params.Path[1:], params.Name), "."),
			}

			fi.Key = strings.ReplaceAll(fi.Key, ".[]", "[]")

			//println(fi.Key)
			if err := refl.PopulateFieldsFromTags(&fi, params.Field.Tag); err != nil {
				return err
			}

			fs.Form = append(fs.Form, fi)

			return nil
		},
	))
	if err != nil {
		return fmt.Errorf("reflecting %s schema: %w", name, err)
	}

	fs.Form = append(fs.Form, FormItem{FormType: "submit", FormTitle: "Submit"})
	fs.Schema = schema
	r.schemas[name] = fs

	return nil
}

func (r *Repository) Schema(name string) FormSchema {
	return r.schemas[name]
}
