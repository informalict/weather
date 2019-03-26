package api

import (
	"database/sql"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
)

type WeatherWebService struct {
	client http.Client
}

type Location struct {
	CityName    string `json:"city_name" description:"name of the city"`
	CountryCode string `json:"country_code" description:"country code"`
	CityId      int    `json:"city_id" description:"identifier of the location in open weather map service"`
}

const (
	weatherApiUrl     = "http://api.openweathermap.org/data"
	weatherApiVersion = "2.5"
	weatherApiToken   = "d7bf62922891f21c84174792d611c5fc"
)

func NewWeatherWebService(client http.Client) *WeatherWebService {
	return &WeatherWebService{
		client: client,
	}
}

func (l *WeatherWebService) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/cities").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"cities"}

	ws.Route(ws.GET("/{city_id}").To(l.getWeather).
		Doc("get a location").
		Param(ws.PathParameter("city_id", "identifier of the city").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Location{}).
		Returns(200, "OK", Location{}).
		Returns(404, "Not Found", nil))

	// TODO should I use PUT or POST here?
	ws.Route(ws.PUT("").To(l.createLocation).
		Doc("create a city").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(Location{}))

	ws.Route(ws.GET("/").To(l.getLocations).
		Doc("get all cities").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]Location{}).
		Returns(200, "OK", []Location{}))

	ws.Route(ws.DELETE("/{city_id}").To(l.removeLocation).
		Doc("delete a city").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("city_id", "identifier of the city").DataType("string")))

	return ws
}

func (l *WeatherWebService) buildWeatherEndpoint(cityId string) string {
	//TODO maybe construct request with params
	//uri := fmt.Sprintf("%s/%s/weather?appid=%s&q=%s", weatherApiUrl, weatherApiVersion, weatherApiToken, city)
	//
	//if len(country) > 0 {
	//	uri += fmt.Sprintf(",%s", country)
	//}
	return fmt.Sprintf("%s/%s/weather?appid=%s&id=%s", weatherApiUrl, weatherApiVersion, weatherApiToken, cityId)
}

func (l *WeatherWebService) getWeather(request *restful.Request, response *restful.Response) {
	cityId := request.PathParameter("city_id")

	resp, err := l.client.Get(l.buildWeatherEndpoint(cityId, ""))
	if err != nil {
		log.Println(err)
		//TODO can be obtained for location and country code
		response.WriteErrorString(http.StatusNotFound, "weather can not be obtained for '%s' city")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("%v", err)
		response.WriteErrorString(http.StatusNotFound, "User could not be found.")
		return
	}
	response.WriteErrorString(http.StatusOK, string(body))
}

func (l *WeatherWebService) getLocations(request *restful.Request, response *restful.Response) {

	//TODO get all locations from database and return to user
	//TODO should I get all locations from api weather i write tha into database or cache?
	list := []Location{
		{
			CityName:    "London",
			CountryCode: "uk",
			CityId:      2,
		},
		{
			CityName:    "Warsaw",
			CountryCode: "pl",
			CityId:      3,
		},
	}
	response.WriteEntity(list)
}

func (l *WeatherWebService) removeLocation(request *restful.Request, response *restful.Response) {
	locationId := request.PathParameter("city_id")
	//TODO delete from database. Do I have to delete weather history for that location?
	response.WriteErrorString(http.StatusOK, fmt.Sprintf("Location '%s' has been deleted", locationId))
}

//curl -vvv -X PUT -H "content-type: application/json" --data '{"city_name": "Warsaw", "country_code": "PL"}' localhost:8080/locations
func (l *WeatherWebService) createLocation(request *restful.Request, response *restful.Response) {
	location := Location{}

	err := request.ReadEntity(&location)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}

	l.testPostgres(location)
	// TODO use regex on parameters
	// TODO do I have to city table?
	// TODO check if location exists in database and if so return an error
	// TODO get location from using city and country code
	// TODO write location to database
	// TODO get weather for created location
	// TODO write weather for created location
	response.WriteHeaderAndEntity(http.StatusCreated, location)
}

const (
	host     = "localhost" //TODO env from docker compose
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "weather"
)

func (l *WeatherWebService) testPostgres(location Location) {
	stringConnection := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", stringConnection)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	// TODO use ORM https://github.com/go-pg/pg
	sqlStatement := `INSERT INTO locations (city, country, city_id) VALUES ($1, $2, $3)`
	_, err = db.Exec(sqlStatement, location.CityName, location.CountryCode, location.CityId)
	if err != nil {
		panic(err)
	}
}
