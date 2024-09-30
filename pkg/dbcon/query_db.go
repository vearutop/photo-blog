package dbcon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/swaggest/usecase"
)

type dbQuery struct {
	Instance  string `json:"instance" enum:"default,stats"`
	Statement string `json:"statement" formType:"textarea" title:"Statement" description:"SQL statement to execute."`
}

func DBQuery(deps Deps) usecase.Interactor {
	type Result struct {
		Statement string          `json:"statement"`
		Columns   []string        `json:"columns"`
		Values    [][]interface{} `json:"values"`
		Elapsed   string          `json:"elapsed"`
		Error     string          `json:"error"`
		Instance  string          `json:"instance"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input dbQuery, output *[]Result) error {
		db := deps.DBInstances()[input.Instance]

		var results []Result

		for _, statement := range strings.Split(input.Statement, "-- next statement") {
			statement = strings.TrimSpace(statement)

			result := Result{
				Statement: statement,
				Instance:  input.Instance,
			}

			start := time.Now()
			rows, err := db.QueryContext(ctx, statement)
			if err != nil {
				result.Error = err.Error()
				results = append(results, result)

				continue
			}
			result.Elapsed = time.Since(start).String()

			cols, _ := rows.Columns()
			defer rows.Close()

			result.Columns = cols

			for rows.Next() {
				values := make([]interface{}, len(cols))
				valuePointers := make([]interface{}, len(cols))
				for i := range values {
					valuePointers[i] = &values[i]
				}

				if err := rows.Scan(valuePointers...); err != nil {
					return fmt.Errorf("scan rows: %w", err)
				}

				for i, v := range values {
					if iv, ok := v.(int64); ok {
						values[i] = strconv.Itoa(int(iv))
					}
				}

				result.Values = append(result.Values, values)
			}

			results = append(results, result)
		}

		*output = results

		return nil
	})

	return u
}
