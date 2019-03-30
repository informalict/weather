package app

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"net/http"
	"strconv"
)

type Location struct {
	CityName    string  `json:"city_name" description:"name of the city"`
	CountryCode string  `json:"country_code" description:"country code"`
	LocationId  int     `json:"location_id" description:"identifier of the location in open weather map service"`
	Latitude    float32 `json:"latitude" description:"name of the city"`
	Longitude   float32 `json:"longitude" description:"name of the city"`
}

type LocationEndpoint struct {
	db                databaseProvider
	openWeatherMapAPI *OpenWeatherAPI
}

func NewLocationEndpoint(db databaseProvider, o *OpenWeatherAPI) *LocationEndpoint {
	return &LocationEndpoint{
		db:                db,
		openWeatherMapAPI: o,
	}
}

func (l *LocationEndpoint) Endpoint() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/locations").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"locations"}

	ws.Route(ws.GET("/{location_id}").To(l.getLocation).
		Doc("get a location").
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Location{}).
		Returns(http.StatusOK, "OK", Location{}).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	ws.Route(ws.POST("").To(l.createLocation).
		Doc("create a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(Location{}).
		Returns(http.StatusCreated, "OK", Location{}).
		Returns(http.StatusBadRequest, "invalid input data", nil).
		Returns(http.StatusGatewayTimeout, "service is unavailable", nil).
		Returns(http.StatusBadGateway, "service is unavailable", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil))

	ws.Route(ws.GET("/").To(l.getLocations).
		Doc("get all locations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]Location{}).
		Returns(http.StatusOK, "OK", []Location{}).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil))

	ws.Route(ws.DELETE("/{location_id}").To(l.deleteLocation).
		Doc("delete a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, "service is unavailable", nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	return ws
}

func (l *LocationEndpoint) getLocation(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	loc, err := l.db.getDBLocation(locationId)
	if err != nil {
		if err == DBNoRows {
			response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("location '%d' not found", locationId))
			return
		}
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	response.WriteEntity(loc)
}

func (l *LocationEndpoint) createLocation(request *restful.Request, response *restful.Response) {
	// TODO check if location exists in database and if so return an error StatusConflict
	location := Location{}
	err := request.ReadEntity(&location)
	if err != nil {
		logger.Error("Create location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "invalid data input")
		return
	}

	s := location.CityName
	if len(location.CountryCode) > 0 {
		s += fmt.Sprintf(",%s", location.CountryCode)
	}

	result, err, status := l.openWeatherMapAPI.getWeather(map[string]string{"q": s})
	if err != nil {
		response.WriteErrorString(status, "service is unavailable")
		return
	}

	location = Location{
		CityName:    result.Name,
		LocationId:  result.Id,
		CountryCode: result.Sys.Country,
		Latitude:    result.Coord.Latitude,
		Longitude:   result.Coord.Longitude,
	}

	err = l.db.saveDBLocation(location)
	if err != nil {
		logger.Error("Create location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, &location)
}

func (l *LocationEndpoint) getLocations(request *restful.Request, response *restful.Response) {
	list, err := l.db.getDBLocations()
	if err != nil {
		logger.Error("Get locations: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, "service is unavailable")
		return
	}

	response.WriteEntity(list)
}

func (l *LocationEndpoint) deleteLocation(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	if err = l.db.deleteDBLocation(id); err != nil {
		if err == DBNoRows {
			response.WriteErrorString(http.StatusNotFound,
				fmt.Sprintf("location '%d' does not exist", id))
			return
		}

		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable,
			fmt.Sprintf("can not delete location '%d'", id))
		return
	}

	response.WriteEntity(nil)
}
