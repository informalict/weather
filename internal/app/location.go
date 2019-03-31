package app

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/google/logger"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

const (
	locationNotFound     = "location '%s' not found"
	locationInvalidID    = "location_id must be an integer"
	serviceIsUnavailable = "service is unavailable"
)

// Location refers to database table 'locations'
type Location struct {
	CityName    string  `json:"city_name" description:"name of the city"`
	CountryCode string  `json:"country_code" description:"country code"`
	LocationID  int     `json:"location_id" description:"identifier of the location in open weather map service"`
	Latitude    float32 `json:"latitude" description:"name of the city"`
	Longitude   float32 `json:"longitude" description:"name of the city"`
}

// LocationEndpoint stores connection to database and open weather API
type LocationEndpoint struct {
	db                databaseProvider
	openWeatherMapAPI *OpenWeatherAPI
}

// NewLocationEndpoint returns LocationEndpoint instance
func NewLocationEndpoint(db databaseProvider, o *OpenWeatherAPI) *LocationEndpoint {
	return &LocationEndpoint{
		db:                db,
		openWeatherMapAPI: o,
	}
}

// Endpoint is a webservice for locations
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
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil).
		Returns(http.StatusNotFound, "location id not found", nil))

	ws.Route(ws.POST("").To(l.createLocation).
		Doc("create a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(Location{}).
		Returns(http.StatusCreated, "OK", Location{}).
		Returns(http.StatusBadRequest, "invalid input data", nil).
		Returns(http.StatusGatewayTimeout, "open weather api timeout", nil).
		Returns(http.StatusBadGateway, "open weather api error", nil).
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil).
		Returns(http.StatusConflict, "location already exist", nil).
		Returns(http.StatusNotFound, "location does not exist", nil))

	ws.Route(ws.GET("/").To(l.getLocations).
		Doc("get all locations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]Location{}).
		Returns(http.StatusOK, "OK", []Location{}).
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil))

	ws.Route(ws.DELETE("/{location_id}").To(l.deleteLocation).
		Doc("delete a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusServiceUnavailable, serviceIsUnavailable, nil).
		Returns(http.StatusNotFound, "location does not exist", nil))

	return ws
}

func (l *LocationEndpoint) getLocation(request *restful.Request, response *restful.Response) {
	locationID, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusBadRequest, locationInvalidID)
		return
	}

	loc, err := l.db.getDBLocation(locationID)
	if err != nil {
		if err == ErrDBNoRows {
			response.WriteErrorString(http.StatusNotFound, fmt.Sprintf(locationNotFound, strconv.Itoa(locationID)))
			return
		}
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	response.WriteEntity(loc)
}

func (l *LocationEndpoint) createLocation(request *restful.Request, response *restful.Response) {
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

	result, status, err := l.openWeatherMapAPI.getWeather(map[string]string{"q": s})
	if err != nil {
		logger.Error("Create location: ", err)
		if status == http.StatusNotFound {
			response.WriteErrorString(status, fmt.Sprintf(locationNotFound, s))
		} else {
			response.WriteErrorString(status, serviceIsUnavailable)
		}
		return
	}

	if _, err = l.db.getDBLocation(result.ID); err == nil {
		str := fmt.Sprintf("location '%s' already exist", s)
		logger.Info("Create location: ", errors.New(str))
		response.WriteErrorString(http.StatusConflict, str)
		return
	}

	location = Location{
		CityName:    result.Name,
		LocationID:  result.ID,
		CountryCode: result.Sys.Country,
		Latitude:    result.Coord.Latitude,
		Longitude:   result.Coord.Longitude,
	}

	err = l.db.saveDBLocation(location)
	if err != nil {
		logger.Error("Create location: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	logger.Info("New location has been created ", location)
	response.WriteHeaderAndEntity(http.StatusCreated, &location)
}

func (l *LocationEndpoint) getLocations(request *restful.Request, response *restful.Response) {
	list, err := l.db.getDBLocations()
	if err != nil {
		logger.Error("Get locations: ", err)
		response.WriteErrorString(http.StatusServiceUnavailable, serviceIsUnavailable)
		return
	}

	if list == nil {
		response.WriteEntity(make([]Location, 0))
		return
	}
	response.WriteEntity(list)
}

func (l *LocationEndpoint) deleteLocation(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusBadRequest, locationInvalidID)
		return
	}

	if err = l.db.deleteDBLocation(id); err != nil {
		if err == ErrDBNoRows {
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
