package main

import (
	"context"
	"errors"
	"github.com/vearutop/photo-blog/pkg/jsonform"
	"log"
	"net/http"

	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v4emb"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	_ "modernc.org/sqlite"
)

type userStatus string

func (us userStatus) Enum() []interface{} {
	return []interface{}{
		"new",
		"approved",
		"active",
		"deleted",
	}
}

// A demo app that receives data from http and stores it in db

type user struct {
	FirstName string     `json:"firstName" required:"true" title:"First name" minLength:"3"`
	LastName  string     `json:"lastName" required:"true" title:"Last name" minLength:"3"`
	Locale    string     `json:"locale" title:"User locale" enum:"ru-RU,en-US"`
	Age       int        `json:"age" title:"Age" minimum:"1"`
	Status    userStatus `json:"status" title:"Status"`
	Bio       string     `json:"bio" title:"Bio" description:"A brief description of the person." formType:"textarea"`
}

func (user) Title() string {
	return "User"
}

func (user) Description() string {
	return "User is a sample entity."
}

type userRepo struct {
	st []user
}

func (r *userRepo) create(u user) {
	r.st = append(r.st, u)
}

func (r *userRepo) update(i int, u user) {
	r.st[i] = u
}

func (r userRepo) list() []user {
	return r.st
}

func (r userRepo) get(i int) (user, error) {
	if i > len(r.st) {
		return user{}, errors.New("user not found")
	}

	return r.st[i], nil
}

func createUser(ur userRepo) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input user, output *struct{}) error {
		ur.create(input)

		return nil
	})
	// Describe use case interactor.
	u.SetTitle("Create User")
	u.SetExpectedErrors(status.InvalidArgument)

	return u
}

func listUsers(ur userRepo) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *[]user) (err error) {
		*output = ur.list()

		return err
	})
	// Describe use case interactor.
	u.SetTitle("List Users")
	u.SetExpectedErrors(status.InvalidArgument)

	return u
}

func main() {
	ur := userRepo{}
	s := web.DefaultService()

	// Init API documentation schema.
	s.OpenAPI.Info.Title = "Nano UI"
	s.OpenAPI.Info.WithDescription("This app showcases a trivial REST API.")
	s.OpenAPI.Info.Version = "v1.2.3"

	// Add use case handler to router.
	s.Post("/user", createUser(ur))
	s.Get("/user", listUsers(ur))

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	jf := jsonform.NewRepository(&s.OpenAPICollector.Reflector().Reflector)
	_ = jf.AddSchema("user", user{})

	jf.Mount(s, "/json-form/")

	s.Method(http.MethodGet, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
<a href="/json-form/form.html?title=Edit%20user&amp;schema=user-schema.json&amp;value=/users/1.json">Edit user</a>

<a href="/json-form/form.html?title=Create%20user&amp;schemaUrl=user-schema.json&amp;submitUrl=/user&amp;submitMethod=POST">Create user</a>

`))
	}))

	// Start server.
	log.Println("SwaggerUI docs at http://localhost:8011/docs")

	if err := http.ListenAndServe("localhost:8011", s); err != nil {
		log.Fatal(err)
	}
}
