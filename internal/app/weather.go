package app

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"net/http"
	"strconv"
)

type StatisticsEndpoint struct {
	db                databaseProvider
	openWeatherMapAPI *OpenWeatherAPI
}

func NewStatisticsEndpoint(db databaseProvider, o *OpenWeatherAPI) *StatisticsEndpoint {
	return &StatisticsEndpoint{
		db:                db,
		openWeatherMapAPI: o,
	}
}

type Statistic struct {
	Temperature float32 `json:"temperature"`
	LocationId  int
	TempMin     float32 `json:"temp_min"`
	TempMax     float32 `json:"temp_max"`
	Type        string  `json:"type"`
}

func (w *StatisticsEndpoint) Endpoint() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/weather").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"weather"}

	ws.Route(ws.GET("/{location_id}").To(w.getWeather).
		Doc("get the weather").
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Statistic{}).
		Returns(http.StatusOK, "OK", Statistic{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	return ws
}

func (w *StatisticsEndpoint) getWeather(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	_, err = w.db.getDBLocation(locationId)
	if err != nil {
		if err == DBNoRows {
			response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("location '%d' not found", locationId))
			return
		}

		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	result, err, status := w.openWeatherMapAPI.getWeather(map[string]string{"id": strconv.Itoa(locationId)})
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(status, "service is unavailable")
		return
	}

	s := Statistic{
		Temperature: result.Main.Temp,
		LocationId:  locationId,
		TempMin:     result.Main.TempMin,
		TempMax:     result.Main.TempMax,
	}

	if len(result.Weather) > 0 {
		s.Type = result.Weather[0].Main // TODO save more then one weather condition
	}

	err = w.db.saveDBStatistics(s)
	if err != nil {
		logger.Error("Save weather: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, &s)
}
