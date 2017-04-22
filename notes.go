package main

import (
	"log"
	"fmt"
	"database/sql"
	"flag"
	"notes/models"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
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
	connectionString := fmt.Sprintf("%s:%s@/%s", *userName, *password, *databaseName)
	log.Println(fmt.Sprintf("ConnectionString: %s", connectionString))
	models.InitDB(connectionString)

	// intentionally not using a router framework since this is intended to be a microservice
	http.HandleFunc("/addNote", routes.AddNote)
	http.HandleFunc("/addTags", routes.AddTags)
	http.HandleFunc("/getActiveNotes", routes.GetActiveNotes)
	http.HandleFunc("/search", routes.Search)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), nil))
}
