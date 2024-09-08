package dbcon

import (
	"github.com/swaggest/rest/web"
)

func Mount(s *web.Service, deps Deps) {
	s.Get("/db.html", DBConsole(deps))
	s.Post("/query-db", DBQuery(deps.DBInstances()))
	s.Get("/query-db.csv", DBQueryCSV(deps.DBInstances()))
}
