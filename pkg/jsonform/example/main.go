package main

import (
	"context"
	"database/sql"
	"github.com/vearutop/photo-blog/pkg/jsonform"
	"log"
	"net/http"

	"github.com/bool64/sqluct"
	"github.com/jmoiron/sqlx"
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
	FirstName string     `formData:"firstName" minLength:"3" db:"first_name"`
	LastName  string     `formData:"lastName" minLength:"3" db:"last_name"`
	Locale    string     `formData:"locale" db:"locale" enum:"ru-RU,en-US"`
	Age       int        `formData:"age" db:"age" minimum:"1"`
	Status    userStatus `formData:"status" db:"status"`
}

type userRepo struct {
	st *sqluct.Storage
}

func (r userRepo) init() {
	if _, err := r.st.DB().Exec(`create table if not exists user(
    first_name VARCHAR(10) NOT NULL,
    last_name VARCHAR(10) NOT NULL,
    age INTEGER NOT NULL,
	locale VARCHAR(10) NOT NULL,
	status VARCHAR(10) NOT NULL
);`); err != nil {
		log.Fatal(err)
	}
}

func (r userRepo) create(ctx context.Context, u user) error {
	_, err := r.st.Exec(ctx, r.st.InsertStmt("user", u))

	return err
}

func (r userRepo) list(ctx context.Context) ([]user, error) {
	var res []user

	q := r.st.SelectStmt("user", user{})

	if err := r.st.Select(ctx, q, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func createUser(ur userRepo) usecase.Interactor {
	u := usecase.NewInteractor[user, struct{}](func(ctx context.Context, input user, output *struct{}) error {
		return ur.create(ctx, input)
	})
	// Describe use case interactor.
	u.SetTitle("Create User")
	u.SetExpectedErrors(status.InvalidArgument)

	return u
}

func listUsers(ur userRepo) usecase.Interactor {
	u := usecase.NewInteractor[struct{}, []user](func(ctx context.Context, input struct{}, output *[]user) (err error) {
		*output, err = ur.list(ctx)

		return err
	})
	// Describe use case interactor.
	u.SetTitle("List Users")
	u.SetExpectedErrors(status.InvalidArgument)

	return u
}

func main() {
	db, err := sql.Open("sqlite", "./db.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	st := sqluct.NewStorage(sqlx.NewDb(db, "sqlite"))
	ur := userRepo{st: st}
	s := web.DefaultService()

	ur.init()

	// Init API documentation schema.
	s.OpenAPI.Info.Title = "DB UI"
	s.OpenAPI.Info.WithDescription("This app showcases a trivial REST API.")
	s.OpenAPI.Info.Version = "v1.2.3"

	// Add use case handler to router.
	s.Post("/user", createUser(ur))
	s.Get("/user", listUsers(ur))

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	jf := jsonform.NewRepository(&s.OpenAPICollector.Reflector().Reflector)
	s.Mount("/json-form", jf.NewHandler())

	// Start server.
	log.Println("SwaggerUI docs at http://localhost:8011/docs")

	if err := http.ListenAndServe("localhost:8011", s); err != nil {
		log.Fatal(err)
	}
}
