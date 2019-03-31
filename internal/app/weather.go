package app

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"net/http"
	"strconv"
)

type WeatherEndpoint struct {
	db                databaseProvider
	openWeatherMapAPI *OpenWeatherAPI
}

func NewWeatherEndpoint(db databaseProvider, o *OpenWeatherAPI) *WeatherEndpoint {
	return &WeatherEndpoint{
		db:                db,
		openWeatherMapAPI: o,
	}
}

type Weather struct {
	TableName   struct{} `sql:"weather" json:"-"`
	Id          int
	Temperature float32 `json:"temperature"`
	LocationId  int
	TempMin     float32     `json:"temp_min"`
	TempMax     float32     `json:"temp_max"`
	Conditions  []Condition `json:"conditions" sql:"-"`
}

type Condition struct {
	StatisticId int `json:"statistic_id"` // pg:"fk:statistic_id"`
	Type        int `json:"type"`
}

type Statistics struct {
	Count            int
	MonthTemperature []MonthTemperatureStatistics
}

type MonthTemperatureStatistics struct {
	TableName struct{} `sql:"weather" json:"-"`
	Min       float32
	Max       float32
	Avg       float32
	Month     string
}

func (w *WeatherEndpoint) Endpoint() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/weather").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"weather"}

	ws.Route(ws.GET("/{location_id}").To(w.getWeather).
		Doc("get the weather").
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Weather{}).
		Returns(http.StatusCreated, "OK", Weather{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	ws.Route(ws.GET("/{location_id}/statistics").To(w.getStatistics).
		Doc("get the weather").
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Weather{}).
		Returns(http.StatusOK, "OK", Weather{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	return ws
}

func (w *WeatherEndpoint) getStatistics(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get statistics: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	s, err := w.db.getStatistics(locationId)
	if err != nil {
		logger.Error("Get statistics: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, &s)
}

func (w *WeatherEndpoint) getWeather(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	_, err = w.db.getDBLocation(locationId)
	if err != nil {
		if err == DBNoRows {
			response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("location '%d' not found", locationId))
			return
		}

		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	result, err, status := w.openWeatherMapAPI.getWeather(map[string]string{"id": strconv.Itoa(locationId)})
	if err != nil {
		logger.Error("Get weather: ", err)
		response.WriteErrorString(status, "service is unavailable")
		return
	}

	s := Weather{
		Temperature: result.Main.Temp,
		LocationId:  locationId,
		TempMin:     result.Main.TempMin,
		TempMax:     result.Main.TempMax,
	}

	for _, v := range result.Description {
		s.Conditions = append(s.Conditions, Condition{
			Type: v.Id,
		})
	}

	err = w.db.saveDBWeather(s)
	if err != nil {
		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, &s)
}
