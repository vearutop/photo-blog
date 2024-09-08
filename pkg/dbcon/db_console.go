package dbcon

import (
	"context"
	"database/sql"
	"net/http"

	jsonform "github.com/swaggest/jsonform-go"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type Deps interface {
	SchemaRepository() *jsonform.Repository
	DBInstances() map[string]*sql.DB
}

// DBConsole creates use case interactor to show DB console.
func DBConsole(deps Deps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *response.EmbeddedSetter) error {
		p := jsonform.Page{}

		p.Title = "DB Console"
		p.AppendHTMLHead = `
<link rel="icon" href="/static/favicon.png" type="image/png"/>
<script
			  src="https://code.jquery.com/jquery-3.7.1.slim.min.js"
			  integrity="sha256-kmHvs0B+OpCW5GVHUNjv9rOmY0IvSIRcf7zGUDTDQM8="
			  crossorigin="anonymous"></script>
<script src="/static/db/script.js?2"></script>
<link rel="stylesheet" href="/static/db/style.css">
`
		p.AppendHTML = `
<div style="margin: 2em">

<a href="#" style="display:none;margin-bottom: 10px" id="dl-csv" class="btn btn-primary" target="_blank">Download CSV</a> <span id="num-rows"></span>
<div id="query-results">

</div>
</div>
`
		return deps.SchemaRepository().Render(out.ResponseWriter(), p,
			jsonform.Form{
				Title:             "Query SQL",
				SubmitURL:         "/query-db",
				SubmitMethod:      http.MethodPost,
				SuccessStatus:     http.StatusOK,
				Value:             dbQuery{},
				OnSuccess:         `onQuerySQLSuccess`,
				OnBeforeSubmit:    `onQuerySQLBeforeSubmit`,
				OnRequestFinished: `onQuerySQLFinished`,
			},
		)
	})

	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
