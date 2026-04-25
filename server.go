package main

import (
	"net/http"
	"notes/routes"
	"strings"

	"github.com/husobee/vestigo"
)

func newRouter() *vestigo.Router {
	router := vestigo.NewRouter()

	// schedules
	router.Post("/schedules", routes.CreateSchedule)
	router.Get("/schedules", routes.ListSchedules)
	router.Get("/schedules/:id", routes.GetSchedule)
	router.Delete("/schedules/:id", routes.DeleteSchedule)

	// audits
	router.Get("/schedules/:id/audits", routes.ListAuditsBySchedule)
	router.Get("/audits/recent/:n", routes.ListRecentAudits)
	router.Get("/audits", routes.ListAuditsByDateRange)

	return router
}

func newAppHandler(uiRoot string) http.Handler {
	api := newRouter()
	ui := http.StripPrefix("/ui", http.FileServer(http.Dir(uiRoot)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/ui":
			http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
		case r.URL.Path == "/ui/" || strings.HasPrefix(r.URL.Path, "/ui/"):
			ui.ServeHTTP(w, r)
		default:
			api.ServeHTTP(w, r)
		}
	})
}
