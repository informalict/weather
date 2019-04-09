package main

import (
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"github.com/mieczyslaw1980/weather/internal/app"

	"time"
)

func main() {
	file, err := os.OpenFile("/tmp/weather.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	logFile := logger.Init("Logger", true, false, file)
	defer logFile.Close()

	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
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

	logger.Info("Weather service start")
	err = http.ListenAndServe(":8080", nil)
	logger.Fatal(err)
}
