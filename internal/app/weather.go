package app

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"net/http"
	"strconv"
)

// Weather refers to database table 'locations'
type Weather struct {
	TableName   struct{} `sql:"weather" json:"-"`
	ID          int      `json:"-"`
	Temperature float32  `json:"temperature"`
	LocationID  int
	TempMin     float32     `json:"temp_min"`
	TempMax     float32     `json:"temp_max"`
	Conditions  []Condition `json:"conditions" sql:"-"`
}

// Condition refers to database table 'conditions'
type Condition struct {
	StatisticID int    `json:"statistic_id"`
	Type        string `json:"type"`
}

// Statistics provides overall statistics
type Statistics struct {
	Count            int
	MonthTemperature []MonthTemperatureStatistics
	DailyCondition   map[string][]string
}

// MonthTemperatureStatistics contains temperature statistics for each month
type MonthTemperatureStatistics struct {
	TableName struct{} `sql:"weather" json:"-"`
	Min       float32
	Max       float32
	Avg       float32
	Month     string
}

// DailyConditionStatistics contains type of weather grouped by type and day
type DailyConditionStatistics struct {
	Type string
	Date string
}

// WeatherEndpoint stores connection to database and open weather API
type WeatherEndpoint struct {
	db                databaseWeatherProvider
	openWeatherMapAPI *OpenWeatherAPI
}

// NewWeatherEndpoint returns WeatherEndpoint instance
func NewWeatherEndpoint(db databaseWeatherProvider, o *OpenWeatherAPI) *WeatherEndpoint {
	return &WeatherEndpoint{
		db:                db,
		openWeatherMapAPI: o,
	}
}

// Endpoint is a webservice for weather statistics
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
		Returns(http.StatusOK, "OK", Weather{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil).
		Returns(http.StatusNotFound, "location does not exist", nil).
		Returns(http.StatusGatewayTimeout, "open weather api timeout", nil).
		Returns(http.StatusBadGateway, "open weather api error", nil))

	ws.Route(ws.GET("/{location_id}/statistics").To(w.getStatistics).
		Doc("get the weather").
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Weather{}).
		Returns(http.StatusOK, "OK", Weather{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil).
		Returns(http.StatusNotFound, "location does not exist", nil))

	return ws
}

func (w *WeatherEndpoint) getStatistics(request *restful.Request, response *restful.Response) {
	locationID, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get statistics: ", err)
		response.WriteErrorString(http.StatusBadRequest, locationInvalidID)
		return
	}

	if _, err := w.db.getDBLocation(locationID); err != nil {
		logger.Error("Get statistics: ", err)
		if err == ErrDBNoRows {
			response.WriteErrorString(http.StatusNotFound,
				fmt.Sprintf("location '%d' does not exist", locationID))
			return
		}
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	s, err := w.db.getStatistics(locationID)
	if err != nil {
		logger.Error("Get statistics: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, &s)
}

func (w *WeatherEndpoint) getWeather(request *restful.Request, response *restful.Response) {
	locationID, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusBadRequest, locationInvalidID)
		return
	}

	_, err = w.db.getDBLocation(locationID)
	if err != nil {
		if err == ErrDBNoRows {
			response.WriteErrorString(http.StatusNotFound, fmt.Sprintf(locationNotFound, strconv.Itoa(locationID)))
			return
		}

		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	result, status, err := w.openWeatherMapAPI.getWeather(map[string]string{"id": strconv.Itoa(locationID)})
	if err != nil {
		logger.Error("Get weather: ", err)
		if status == http.StatusNotFound {
			response.WriteErrorString(status, fmt.Sprintf(locationNotFound, strconv.Itoa(locationID)))
		} else {
			response.WriteErrorString(status, serviceIsUnavailable)
		}
		return
	}

	s := Weather{
		Temperature: result.Main.Temp,
		LocationID:  locationID,
		TempMin:     result.Main.TempMin,
		TempMax:     result.Main.TempMax,
	}

	for _, v := range result.Description {
		s.Conditions = append(s.Conditions, Condition{
			Type: v.Main,
		})
	}

	err = w.db.saveDBWeather(s)
	if err != nil {
		logger.Error("Get weather: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, &s)
}
