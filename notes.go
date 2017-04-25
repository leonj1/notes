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
)

type Env struct {
	db *sql.DB
}

func main() {
	var userName = flag.String("user", "", "db username")
	var password = flag.String("pass", "", "db password")
	var databaseName = flag.String("db", "", "db name")
	var serverPort = flag.String("port", "", "server port")
	flag.Parse()

	// open connection to db
	connectionString := fmt.Sprintf("%s:%s@/%s?parseTime=true", *userName, *password, *databaseName)
	models.InitDB(connectionString)

	router := vestigo.NewRouter()

	router.Get("/notes", routes.ActiveNotes)
	router.Post("/notes", routes.AddNote)
	router.Put("/notes/:id", routes.AddTags)
	router.Delete("/notes/:id", routes.DeleteNote)

	log.Println("Starting web server")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), router))
}
