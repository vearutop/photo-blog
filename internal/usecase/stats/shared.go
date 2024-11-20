package stats

import (
	"github.com/swaggest/refl"
	"reflect"
	"strings"
)

type pageData struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Tables      []Table `json:"tables"`
}

type Table struct {
	Title string      `json:"title"`
	Rows  interface{} `json:"rows"`
}

type infoRow struct {
	Column string      `json:"column"`
	Value  interface{} `json:"value"`
}

func infoRows(v any) []infoRow {
	var rows []infoRow
	refl.WalkTaggedFields(reflect.ValueOf(v), func(v reflect.Value, sf reflect.StructField, tag string) {
		rows = append(rows, infoRow{
			Column: strings.TrimRight(tag, "."),
			Value:  v.Interface(),
		})
	}, "description")

	return rows
}
