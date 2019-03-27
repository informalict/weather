package api

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-pg/pg"
	"strconv"

	//"github.com/go-pg/pg/orm"
	"github.com/google/logger"
	//_ "github.com/lib/pq"
	"io/ioutil"
	"net/http"
)

type WeatherLocation struct {
	client http.Client
}

type Location struct {
	CityName    string  `json:"city_name" description:"name of the city"`
	CountryCode string  `json:"country_code" description:"country code"`
	LocationId  int     `json:"location_id" description:"identifier of the location in open weather map service"`
	Latitude    float32 `json:"latitude" description:"name of the city"`
	Longitude   float32 `json:"longitude" description:"name of the city"`
}

type OpenMapWeather struct {
	Coord OpenMapWeatherCoordinates `json:"coord"`
	Name  string                    `json:"name"`
	Id    int                       `json:"id"`
	Sys   OpenMapWeatherSys         `json:sys`
}

type OpenMapWeatherCoordinates struct {
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
}

type OpenMapWeatherSys struct {
	Country string `json:"country"`
}

const (
	weatherApiUrl     = "http://api.openweathermap.org/data"
	weatherApiVersion = "2.5"
	weatherApiToken   = "d7bf62922891f21c84174792d611c5fc"
)

func NewWeatherWebService(client http.Client) *WeatherLocation {
	return &WeatherLocation{
		client: client,
	}
}

func (l *WeatherLocation) WebService() *restful.WebService {
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
		Returns(http.StatusInternalServerError, "database does not work properly", nil)))

	return ws
}

func (l *WeatherLocation) getLocation(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	location, err := l.getDBLocation(locationId)
	if err != nil {
		logger.Error("Get location: ", err)
		response.WriteErrorString(http.StatusInternalServerError, "Service is unavailable")
		return
	}

	response.WriteEntity(location)
}

//curl -vvv -X PUT -H "content-type: application/json" --data '{"city_name": "Warsaw", "country_code": "PL"}' localhost:8080/locations
func (l *WeatherLocation) createLocation(request *restful.Request, response *restful.Response) {
	location := Location{}
	// TODO validate (regex)?
	// TODO check if location exists in database and if so return an error
	err := request.ReadEntity(&location)
	if err != nil {
		logger.Error("Create location: ", err)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, err := l.client.Get(l.buildWeatherEndpoint1(location.CityName, location.CountryCode))
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

	location.CityName = result.Name
	location.LocationId = result.Id
	location.CountryCode = result.Sys.Country
	location.Latitude = result.Coord.Latitude
	location.Longitude = result.Coord.Longitude

	err = l.saveDBLocation(location)
	if err != nil {
		logger.Error("Create location: ", err)
		// TODO StatusConflict if exists record
		response.WriteHeaderAndEntity(http.StatusInternalServerError, location)
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, location)
}

func (l *WeatherLocation) getLocations(request *restful.Request, response *restful.Response) {
	list, err := l.getDBLocations()
	if err != nil {
		logger.Error("Get locations: ", err)
		response.WriteErrorString(http.StatusInternalServerError, "Service is unavailable")
		return
	}

	response.WriteEntity(list)
}

func (l *WeatherLocation) deleteLocation(request *restful.Request, response *restful.Response) {
	locationId, err := strconv.Atoi(request.PathParameter("location_id"))
	if err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusBadRequest, "location_id must be an integer")
		return
	}

	if err = l.deleteDBLocation(locationId); err != nil {
		logger.Error("Delete location: ", err)
		response.WriteErrorString(http.StatusInternalServerError,
			fmt.Sprintf("can not delete id location '%d'", locationId))
		return
	}

	// TODO delete also history or ON DELETE CASCADE
	response.WriteErrorString(http.StatusOK, fmt.Sprintf("Location '%s' has been deleted", locationId))
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
func (l *WeatherLocation) saveDBLocation(location Location) error {
	db := pg.Connect(&pg.Options{
		User:     "postgres",       //todo env
		Database: "weather",        //todo env
		Password: "postgres",       //todo env
		Addr:     "localhost:5432", //todo env
	})
	defer db.Close()

	return db.Insert(&location)
}

func (l *WeatherLocation) deleteDBLocation(locationId int) error {
	db := pg.Connect(&pg.Options{
		User:     "postgres",       //todo env
		Database: "weather",        //todo env
		Password: "postgres",       //todo env
		Addr:     "localhost:5432", //todo env
	})
	defer db.Close()

	location := &Location{LocationId: locationId}

	_, err := db.Model(location).Where("location_id = ?location_id").Delete()
	return err
}

func (l *WeatherLocation) getDBLocations() ([]Location, error) {
	db := pg.Connect(&pg.Options{
		User:     "postgres",       //todo env
		Database: "weather",        //todo env
		Password: "postgres",       //todo env
		Addr:     "localhost:5432", //todo env
	})
	defer db.Close()

	locations := []Location{}
	err := db.Model(&locations).Order("country_code ASC", "city_name ASC").Select()
	return locations, err
}

func (l *WeatherLocation) getDBLocation(locationId int) (*Location, error) {
	db := pg.Connect(&pg.Options{
		User:     "postgres",       //todo env
		Database: "weather",        //todo env
		Password: "postgres",       //todo env
		Addr:     "localhost:5432", //todo env
	})
	defer db.Close()

	location := &Location{}
	err := db.Model(location).Where("location_id = ?", locationId).Select()
	if err != nil {
		return nil, err
	}
	return location, nil
}

func (l *WeatherLocation) buildWeatherEndpoint1(cityName, countryCode string) string {
	//TODO maybe construct request with params
	uri := fmt.Sprintf("%s/%s/weather?appid=%s&q=%s", weatherApiUrl, weatherApiVersion, weatherApiToken, cityName)

	if len(countryCode) > 0 {
		uri += fmt.Sprintf(",%s", countryCode)
	}
	return uri
}

func (l *WeatherLocation) buildWeatherEndpoint(cityId string) string {
	//TODO maybe construct request with params
	//uri := fmt.Sprintf("%s/%s/weather?appid=%s&q=%s", weatherApiUrl, weatherApiVersion, weatherApiToken, city)
	//
	//if len(country) > 0 {
	//	uri += fmt.Sprintf(",%s", country)
	//}
	return fmt.Sprintf("%s/%s/weather?appid=%s&id=%s", weatherApiUrl, weatherApiVersion, weatherApiToken, cityId)
}
