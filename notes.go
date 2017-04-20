package main

import (
	"log"
	"fmt"
	"database/sql"
	"flag"
	"net/http"
	"notes/models"
	"github.com/julienschmidt/httprouter"
	_ "github.com/go-sql-driver/mysql"
)

type Env struct {
	db *sql.DB
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
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

	//env := &Env{db: db}

	log.Println(fmt.Sprintf("Listening on port: %s", *serverPort))

	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), router))
}
