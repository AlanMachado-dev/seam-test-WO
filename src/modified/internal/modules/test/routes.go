package test

import (
	"net/http"

	"github.com/ha1tch/seam-ui/internal/ui"
)

func MountRoutes(router *ui.Router) func(mux *http.ServeMux) {
	return func(mux *http.ServeMux) {
		router.MountTestsRoutes(mux)
	}
}
