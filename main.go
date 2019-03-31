package main

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"github.com/mieczyslaw1980/weather/internal/app"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{
		Timeout: time.Duration(4 * time.Second),
	}
	externalAPI, err := app.NewOpenWeatherAPI(client)
	if err != nil {
		logger.Error(err)
		return
	}

	db, err := app.NewDB()
	if err != nil {
		logger.Error(err)
		return
	}

	l := app.NewLocationEndpoint(db, externalAPI)
	w := app.NewWeatherEndpoint(db, externalAPI)

	restful.DefaultContainer.Add(l.Endpoint())
	restful.DefaultContainer.Add(w.Endpoint())

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
	}

	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	logger.Info("Weather service start listening on localhost:8080")
	err = http.ListenAndServe(":8080", nil)
	logger.Fatal(err)
}
