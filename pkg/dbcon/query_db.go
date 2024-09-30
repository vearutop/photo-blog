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

func SplitStatements(s string) []string {
	var (
		quoteChars  = [...]int32{'"', '\'', '`'}
		quoteOpened int32
	)

	prevStart := 0
	prevQuot := false

	// -- Line comment.
	lineCommentStarted := false
	prevDash := false

	// /* Block comment */
	blockCommentStarted := false
	prevSlash := false
	prevAsterisk := false

	var res []string

	for i, c := range s {
		if blockCommentStarted && c != '*' && c != '/' {
			continue
		}

		if lineCommentStarted {
			if c == '\n' {
				lineCommentStarted = false
				continue
			} else {
				continue
			}
		}

		if quoteOpened != 0 {
			// This may be a closing quote or an escaped quot if it is immediately followed by another same quot.
			if c == quoteOpened {
				if prevQuot {
					prevQuot = false
					continue
				}

				prevQuot = true
				continue
			}

			if prevQuot {
				prevQuot = false
				quoteOpened = 0
			}
		}

		if quoteOpened == 0 {
			for _, q := range quoteChars {
				if c == q {
					quoteOpened = q
				}
			}

			if quoteOpened != 0 {
				continue
			}

			// Might be a line comment.
			if c == '-' {
				if prevDash {
					prevDash = false
					lineCommentStarted = true
					continue
				}

				prevDash = true
			} else {
				prevDash = false
			}

			if c == '/' {
				if prevAsterisk && blockCommentStarted {
					blockCommentStarted = false
					prevAsterisk = false
					continue
				}

				prevSlash = true
				continue
			}

			if c == '*' {
				if prevSlash && !blockCommentStarted {
					blockCommentStarted = true
					prevSlash = false
					continue
				}

				prevAsterisk = true
				continue
			}

			prevSlash = false
			prevAsterisk = false

			// Not in an enquoted string, so that's a statement separator.
			if c == ';' {
				st := strings.TrimSpace(s[prevStart:i])
				if len(st) > 0 {
					res = append(res, st)
				}
				prevStart = i + 1
			}
		}
	}

	st := strings.TrimSpace(s[prevStart:])
	if len(st) > 0 {
		res = append(res, st)
	}

	return res
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

		for _, statement := range SplitStatements(input.Statement) {
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
