package main

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/mieczyslaw1980/weather/internal/app"
	"log"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{
		Timeout: time.Duration(3 * time.Second),
	}
	externalAPI := app.NewOpenWeatherAPI(client)
	db := app.NewDB()

	l := app.NewLocationEndpoint(db, externalAPI)
	w := app.NewWeatherEndpoint(db, externalAPI)

	restful.DefaultContainer.Add(l.Endpoint())
	restful.DefaultContainer.Add(w.Endpoint())

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
	}

	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	log.Printf("Weather service start listening on localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
