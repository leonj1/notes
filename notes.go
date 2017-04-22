package main

import (
	"log"
	"fmt"
	"database/sql"
	"flag"
	"notes/models"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"encoding/json"
)

type Env struct {
	db *sql.DB
}

const ContentType = "Content-Type"
const JSON = "application/json"

func foo(w http.ResponseWriter, r *http.Request) {
	var n models.Note
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&n)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	js, err := json.Marshal(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
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

	http.HandleFunc("/", foo)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), nil))
}
