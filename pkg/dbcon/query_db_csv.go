package dbcon

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/swaggest/usecase"
)

func DBQueryCSV(deps Deps) usecase.Interactor {
	type request struct {
		Instance  string `query:"instance" enum:"default,stats"`
		Statement string `query:"statement" formType:"textarea" title:"Statement" description:"SQL Statement to execute."`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input request, output *usecase.OutputWithEmbeddedWriter) error {
		rows, err := deps.DBInstances()[input.Instance].QueryContext(ctx, input.Statement)
		if err != nil {
			return err
		}

		cols, _ := rows.Columns()
		defer rows.Close()

		rw := output.Writer.(http.ResponseWriter)
		rw.Header().Set("Content-Type", "text/csv")
		rw.Header().Set("Content-Disposition", "attachment; filename=\"data.csv\"")
		rw.Header().Set("Content-Transfer-Encoding", "binary")
		w := csv.NewWriter(output.Writer)

		_ = w.Write(cols)

		for rows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				return fmt.Errorf("scan rows: %w", err)
			}

			var values []string

			for i := range cols {
				v := columns[i]

				if iv, ok := v.(int64); ok {
					v = strconv.Itoa(int(iv))
				}

				j, err := json.Marshal(v)
				if err != nil {
					return err
				}

				values = append(values, strings.Trim(string(j), `"`))
			}

			_ = w.Write(values)
		}

		w.Flush()

		return nil
	})

	return u
}
