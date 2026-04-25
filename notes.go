package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"notes/models"
	"notes/services"

	_ "github.com/go-sql-driver/mysql"
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

	// start background scheduler worker
	services.StartWorker()

	log.Println("Starting web server")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), newAppHandler("./ui")))
}
