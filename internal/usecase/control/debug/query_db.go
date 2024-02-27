package debug

import (
	"context"
	"fmt"
	"strconv"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type dbQuery struct {
	Statement string `json:"statement" formType:"textarea" title:"Statement" description:"SQL statement to execute."`
}

func DBQuery(deps *service.Locator) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input dbQuery, output *[]map[string]interface{}) error {
		rows, err := deps.Storage.DB().QueryContext(ctx, input.Statement)
		if err != nil {
			return err
		}

		var out []map[string]interface{}
		cols, _ := rows.Columns()
		defer rows.Close()

		for rows.Next() {
			data := make(map[string]interface{})
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				return fmt.Errorf("scan rows: %w", err)
			}

			for i, colName := range cols {
				v := columns[i]
				if i, ok := v.(int64); ok {
					v = strconv.Itoa(int(i))
				}

				data[colName] = v
			}

			out = append(out, data)
		}

		*output = out

		return nil
	})

	return u
}
