package main

import (
	"log"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"github.com/husobee/vestigo"
	"flag"
	"fmt"
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

	router.Post("/tags", routes.AddTags)
	router.Post("/notes", routes.AddNote)

	router.Delete("/notes/:id", routes.DeleteNote)

	// Catch-All methods to allow easy migration from http.ServeMux
	router.HandleFunc("/general", GeneralHandler)

	log.Println("Starting web server")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), router))
}

func GeneralHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Gotta catch em all!"))
}
