package main

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	//"github.com/go-pg/pg"
	"github.com/mieczyslaw1980/weather/api"
	"log"
	"net/http"
	//"os"
)

func main() {
	db := api.NewDB()
	l := api.NewLocation(db)
	restful.DefaultContainer.Add(l.WebService())

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
	}

	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	log.Printf("Weather service start listening on localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
