// Package ui provides application web user interface.
package ui

import (
	"net/http"
)

// Album serves index page of the application.
func Album() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Static.ServeHTTP(w, r)
	})
}
