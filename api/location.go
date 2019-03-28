package api

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-pg/pg"
	"github.com/google/logger"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Location struct {
	CityName    string  `json:"city_name" description:"name of the city"`
	CountryCode string  `json:"country_code" description:"country code"`
	LocationId  int     `json:"location_id" description:"identifier of the location in open weather map service"`
	Latitude    float32 `json:"latitude" description:"name of the city"`
	Longitude   float32 `json:"longitude" description:"name of the city"`
	client      http.Client
	db          DatabaseLocation
	config      struct {
		database          *pg.Options
		openWeatherMapUrl string
	}
}

type OpenMapWeather struct {
	Coord struct {
		Latitude  float32 `json:"lat"`
		Longitude float32 `json:"lon"`
	} `json:"coord"`
	Name string `json:"name"`
	Id   int    `json:"id"`
	Sys  struct {
		Country string `json:"country"`
	} `json:sys`
}

const (
	weatherApiUrl     = "http://api.openweathermap.org/data"
	weatherApiVersion = "2.5"
)

func NewLocation(dl DatabaseLocation) *Location {
	return &Location{
		client: http.Client{ //TODO should i chnge that
			Timeout: time.Duration(3 * time.Second),
		},
		db: dl,
		config: struct {
			database          *pg.Options
			openWeatherMapUrl string
		}{
			database: &pg.Options{
				User:     os.Getenv("DB_USER"),     //"postgres",       //todo env
				Database: os.Getenv("DB_DATABASE"), //"weather",            //todo env
				Password: os.Getenv("DB_PASSWORD"), //"postgres",               //todo env
				Addr:     os.Getenv("DB_ADDRESS"),  //"localhost:5432",         //todo env
			},
			openWeatherMapUrl: fmt.Sprintf("%s/%s/weather?appid=%s",
				weatherApiUrl, weatherApiVersion, os.Getenv("OPEN_WEATHER_MAP_TOKEN")),
		},
	}
}

func (l *Location) WebService() *restful.WebService {
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
		Returns(http.StatusInternalServerError, "database does not work properly", nil))

	ws.Route(ws.POST("").To(l.createLocation).
		Doc("create a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(Location{})) //TODO returns

	ws.Route(ws.GET("/").To(l.getLocations).
		Doc("get all locations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]Location{}).
		Returns(http.StatusOK, "OK", []Location{}).
		Returns(http.StatusInternalServerError, "database does not work properly", nil))

	ws.Route(ws.DELETE("/{location_id}").To(l.deleteLocation).
		Doc("delete a location").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("location_id", "identifier of the location").DataType("integer")).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusBadRequest, "id location must be an integer", nil).
		Returns(http.StatusInternalServerError, "database does not work properly", nil))

	return ws
}

func (l *Location) getLocation(request *restful.Request, response *restful.Response) {
	var err error

	l.LocationId, err = strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	loc, err := l.db.getDBLocation(l.LocationId)
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusInternalServerError, "Service is unavailable")
		return
	}

	response.WriteEntity(loc)
}

func (l *Location) createLocation(request *restful.Request, response *restful.Response) {
	// TODO validate (regex)?
	// TODO check if location exists in database and if so return an error
	err := request.ReadEntity(l)
	if err != nil {
		logger.Error("Create location: ", err)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, err := l.client.Get(l.buildLocationEndpoint())
	if err != nil {
		logger.Error("Create location: ", err)
		//TODO which status to result?
		response.WriteErrorString(http.StatusBadRequest, "weather can not be obtained for '%s' city")
		return
	}
	defer resp.Body.Close()

	result, err := parseOpenMapWeatherResponse(resp)
	if err != nil {
		logger.Error("Create location: ", err)
		//TODO which status to result?
		response.WriteErrorString(http.StatusInternalServerError, "weather can not be obtained for '%s' city")
		return
	}

	l.CityName = result.Name
	l.LocationId = result.Id
	l.CountryCode = result.Sys.Country
	l.Latitude = result.Coord.Latitude
	l.Longitude = result.Coord.Longitude

	err = l.saveDBLocation()
	if err != nil {
		logger.Error("Create location: ", err)
		// TODO StatusConflict if exists record
		response.WriteHeaderAndEntity(http.StatusInternalServerError, l)
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, l)
}

func (l *Location) getLocations(request *restful.Request, response *restful.Response) {
	list, err := l.db.getDBLocations()
	if err != nil {
		logger.Error("Get locations: ", err)
		response.WriteErrorString(http.StatusInternalServerError, "Service is unavailable")
		return
	}

	response.WriteEntity(list)
}

func (l *Location) deleteLocation(request *restful.Request, response *restful.Response) {
	var err error

	l.LocationId, err = strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	if err = l.deleteDBLocation(); err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusInternalServerError,
			fmt.Sprintf("can not delete id location '%d'", l.LocationId))
		return
	}

	// TODO delete also history or ON DELETE CASCADE
	response.WriteErrorString(http.StatusOK, fmt.Sprintf("Location '%d' has been deleted", l.LocationId))
}

func parseOpenMapWeatherResponse(response *http.Response) (*OpenMapWeather, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	weather := &OpenMapWeather{}
	if err = json.Unmarshal(body, weather); err != nil {
		return nil, err
	}
	return weather, nil
}

//func (l *WeatherLocation) getWeather(request *restful.Request, response *restful.Response) {
//	cityId := request.PathParameter("city_id")
//
//	resp, err := l.client.Get(l.buildWeatherEndpoint(cityId, ""))
//	if err != nil {
//		log.Println(err)
//		//TODO can be obtained for location and country code
//		response.WriteErrorString(http.StatusNotFound, "weather can not be obtained for '%s' city")
//		return
//	}
//	defer resp.Body.Close()
//
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		log.Fatalf("%v", err)
//		response.WriteErrorString(http.StatusNotFound, "User could not be found.")
//		return
//	}
//	response.WriteErrorString(http.StatusOK, string(body))
//}

/*
CREATE TABLE locations (location_id INTEGER PRIMARY KEY, city_name VARCHAR NOT NULL,
country_code CHAR(10) NOT NULL, latitude numeric(6,2), longitude numeric(6,2), UNIQUE(city_name, country_code));
*/
func (l *Location) saveDBLocation() error {
	db := pg.Connect(l.config.database)
	defer db.Close()

	return db.Insert(l)
}

func (l *Location) deleteDBLocation() error {
	db := pg.Connect(l.config.database) //TODO nil
	defer db.Close()

	_, err := db.Model(l).Where("location_id = ?location_id").Delete()
	return err
}

func (l *Location) buildLocationEndpoint() string {
	if l.LocationId > 0 {
		return fmt.Sprintf("%s&id=%d", l.config.openWeatherMapUrl, l.LocationId)
	}

	if len(l.CityName) > 0 {
		uri := fmt.Sprintf("%s&q=%s", l.config.openWeatherMapUrl, l.CityName)
		if len(l.CountryCode) > 0 {
			uri += fmt.Sprintf(",%s", l.CountryCode)
		}
		return uri
	}
	return "" // TODO error?
}
