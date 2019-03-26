package main

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/mieczyslaw1980/weather/api"
	"net/http"
	"time"

	"log"
)

func main() {
	client := http.Client{
		Timeout: time.Duration(3 * time.Second),
	}

	l := api.NewWeatherWebService(client)
	restful.DefaultContainer.Add(l.WebService())

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
	}

	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	log.Printf("Weather service start listening on localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
