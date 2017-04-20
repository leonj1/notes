package main

import (
	"log"
	"fmt"
	"database/sql"
	"flag"
	"notes/models"
	"github.com/plimble/ace"
	_ "github.com/go-sql-driver/mysql"
)

type Env struct {
	db *sql.DB
}

func PostNote(c *ace.C) {
	id := c.Param("id")
	name := c.Param("name")
	age := c.Request.PostFormValue("age")

	data := struct{
		Id string `json:"id"`
		Name string `json:"name"`
		Age string `json:"age"`
	}{
		Id: id,
		Name: name,
		Age: age,
	}
	c.JSON(200, data)
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

	a := ace.New()

	a.GET("/:name", func(c *ace.C) {
		name := c.Param("name")
		c.JSON(200, map[string]string{"hello": name})
	})

	a.POST("/form/:id/:name", PostNote)

	a.Run(fmt.Sprintf(":%s", *serverPort))
}
