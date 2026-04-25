package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/husobee/vestigo"
	"log"
	"net/http"
	"notes/models"
	"notes/routes"
	"notes/services"
)

type Env struct {
	db *sql.DB
}

func main() {
	var userName = flag.String("user", "", "db username")
	var password = flag.String("pass", "", "db password")
	var databaseName = flag.String("db", "", "db name")
	var host = flag.String("host", "127.0.0.1:3306", "db host:port")
	var serverPort = flag.String("port", "", "server port")
	flag.Parse()

	// open connection to db
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", *userName, *password, *host, *databaseName)
	models.InitDB(connectionString)

	router := vestigo.NewRouter()

	router.Get("/notes", routes.AllNotes)
	router.Post("/notes", routes.AddNote)
	router.Put("/notes/:id", routes.AddTags)
	router.Delete("/notes/:id", routes.DeleteNote)

	// common queries
	router.Get("/activenotes", routes.ActiveNotes)

	// filters
	router.Get("/tags/:key/:value", routes.FilterNotesByTag)

	// schedules
	router.Post("/schedules", routes.CreateSchedule)
	router.Get("/schedules", routes.ListSchedules)
	router.Get("/schedules/:id", routes.GetSchedule)
	router.Delete("/schedules/:id", routes.DeleteSchedule)

	// audits
	router.Get("/schedules/:id/audits", routes.ListAuditsBySchedule)
	router.Get("/audits/recent/:n", routes.ListRecentAudits)
	router.Get("/audits", routes.ListAuditsByDateRange)

	// start background scheduler worker
	services.StartWorker()

	log.Println("Starting web server")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), router))
}
