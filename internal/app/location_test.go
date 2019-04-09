package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLocation(t *testing.T) {
	// Arrange
	tests := []struct {
		name          string
		expectedError error
		locationID    string
		db            fakeDatabase
		HTTPStatus    int
	}{
		{
			name:          "Invalid location_id",
			locationID:    "invalid",
			expectedError: errors.New(locationInvalidID),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:       "No connection to database",
			locationID: "123",
			db: fakeDatabase{
				err: errors.New("can not connect to database"),
			},
			expectedError: errors.New(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "Location not found in database",
			locationID: "462356",
			db: fakeDatabase{
				err: sql.ErrNoRows,
			},
			HTTPStatus:    http.StatusNotFound,
			expectedError: errors.New("location '462356' not found"),
		},
		{
			name:       "Get valid location",
			locationID: "123",
			db: fakeDatabase{
				locations: []Location{
					{
						LocationID:  123,
						CityName:    "Warsaw",
						CountryCode: "PL",
						Longitude:   21.01,
						Latitude:    52.23,
					},
				},
			},
			HTTPStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			l := NewLocationEndpoint(test.db, nil)
			request := restful.NewRequest(nil)
			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)
			params := request.PathParameters()
			params["location_id"] = test.locationID

			// Act
			l.getLocation(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
				return
			}

			assert.Nil(t, response.Error())
			res := response.ResponseWriter.(*httptest.ResponseRecorder)
			if assert.NotNil(t, res) {
				lr := Location{}
				err := json.Unmarshal(res.Body.Bytes(), &lr)
				assert.Nil(t, err)
				assert.Equal(t, test.db.locations[0], lr)
			}
		})
	}
}

func TestDeleteLocation(t *testing.T) {
	// Arrange
	tests := []struct {
		name          string
		expectedError error
		locationID    string
		db            fakeDatabase
		HTTPStatus    int
	}{
		{
			name:          "Invalid location_id",
			locationID:    "invalid",
			expectedError: errors.New(locationInvalidID),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:       "No entry in database",
			locationID: "123",
			db: fakeDatabase{
				err: sql.ErrNoRows,
			},
			expectedError: errors.New("location '123' does not exist"),
			HTTPStatus:    http.StatusNotFound,
		},
		{
			name:       "Database error",
			locationID: "123",
			db: fakeDatabase{
				err: errors.New("database error"),
			},
			expectedError: errors.New("can not delete location '123'"),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "Location has been deleted",
			locationID: "123",
			HTTPStatus: http.StatusOK,
			db: fakeDatabase{
				err: nil,
			},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			l := NewLocationEndpoint(test.db, nil)
			request := restful.NewRequest(nil)
			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)
			params := request.PathParameters()
			params["location_id"] = test.locationID

			// Act
			l.deleteLocation(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
			} else {
				assert.Nil(t, response.Error())
			}
		})
	}
}

func TestGetLocations(t *testing.T) {
	// Arrange
	tests := []struct {
		name          string
		expectedError error
		db            fakeDatabase
		HTTPStatus    int
	}{
		{
			name: "Database error",
			db: fakeDatabase{
				err: errors.New("database error"),
			},
			expectedError: errors.New(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "The empty list",
			HTTPStatus: http.StatusOK,
			db: fakeDatabase{
				locations: nil,
			},
		},
		{
			name:       "The list with two locations",
			HTTPStatus: http.StatusOK,
			db: fakeDatabase{
				locations: []Location{
					{
						LocationID:  123,
						CityName:    "Warsaw",
						CountryCode: "PL",
						Longitude:   21.01,
						Latitude:    52.23,
					},
					{
						LocationID:  2643743,
						CityName:    "London",
						CountryCode: "GB",
						Longitude:   -0.13,
						Latitude:    51.51,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			l := NewLocationEndpoint(test.db, nil)
			request := restful.NewRequest(nil)
			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)

			// Act
			l.getLocations(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
				return
			}
			assert.Nil(t, response.Error())
			res := response.ResponseWriter.(*httptest.ResponseRecorder)
			if assert.NotNil(t, res) {
				if test.db.locations == nil {
					test.db.locations = make([]Location, 0) //because it should return empty list
				}
				lr := make([]Location, 2)
				err := json.Unmarshal(res.Body.Bytes(), &lr)
				assert.Nil(t, err)
				assert.Equal(t, test.db.locations, lr)
			}
		})
	}
}

func TestCreateLocation(t *testing.T) {
	// Arrange
	type ExternalAPI struct {
		response   string
		HTTPStatus int
		Timeout    time.Duration
	}
	tests := []struct {
		name          string
		expectedError error
		cityName      string
		countryCode   string
		db            fakeDatabase
		HTTPStatus    int
		externalAPI   ExternalAPI
		contentType   string
	}{
		{
			name:          "Bad request",
			cityName:      "",
			expectedError: fmt.Errorf("invalid data input"),
			HTTPStatus:    http.StatusBadRequest,
			contentType:   "application/invalid",
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Bad request",
			cityName:      "",
			expectedError: fmt.Errorf("input data field 'city_name' is required"),
			HTTPStatus:    http.StatusBadRequest,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Open weather API timeout",
			cityName:      "Warsaw",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusGatewayTimeout,
			externalAPI: ExternalAPI{
				Timeout:    time.Second * 2,
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Invalid format data from open map weather service",
			cityName:      "Warsaw",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusBadGateway,
			externalAPI: ExternalAPI{
				response:   `{invalid json}`,
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Location does not exist in open weather map service",
			cityName:      "Invalid",
			expectedError: fmt.Errorf("location 'Invalid' not found"),
			HTTPStatus:    http.StatusNotFound,
			externalAPI: ExternalAPI{
				response:   `{ "cod": "404", "message": "city not found" }`,
				HTTPStatus: http.StatusNotFound,
			},
		},
		{
			name:          "Location can not be saved in database",
			cityName:      "Warsaw",
			countryCode:   "PL",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
			externalAPI: ExternalAPI{
				response:   `{"city_name": "Warsaw", "country_code": "PL"}`,
				HTTPStatus: http.StatusOK,
			},
			db: fakeDatabase{
				errSave: fmt.Errorf("cannot connect to database"),
				err:     sql.ErrNoRows,
			},
		},
		{
			name:          "Location already exists in database",
			cityName:      "Warsaw",
			countryCode:   "PL",
			expectedError: fmt.Errorf("location 'Warsaw,PL' already exist"),
			HTTPStatus:    http.StatusConflict,
			externalAPI: ExternalAPI{
				response:   `{"city_name": "Warsaw", "country_code": "PL"}`,
				HTTPStatus: http.StatusOK,
			},
			db: fakeDatabase{
				err: nil,
			},
		},
		{
			name:       "Location has been saved",
			cityName:   "Warsaw",
			HTTPStatus: http.StatusCreated,
			externalAPI: ExternalAPI{
				response:   `{ "id": 756135, "name": "Warsaw", "sys": { "country": "PL" } }`,
				HTTPStatus: http.StatusOK,
			},
			db: fakeDatabase{
				err: sql.ErrNoRows,
				locations: []Location{
					{
						LocationID:  756135,
						CityName:    "Warsaw",
						CountryCode: "PL",
					},
				},
			},
		},
	}

	URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
	URLToken := os.Getenv("OPEN_WEATHER_MAP_TOKEN")
	defer func() {
		os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", URLToken)
	}()
	os.Setenv("OPEN_WEATHER_MAP_TOKEN", "token")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if test.externalAPI.Timeout > 0 {
					time.Sleep(test.externalAPI.Timeout) //timeout simulation
				}
				rw.WriteHeader(test.externalAPI.HTTPStatus)
				rw.Header()
				rw.Write([]byte(test.externalAPI.response))
			}))
			defer server.Close()

			client := server.Client()
			client.Timeout = time.Second

			os.Setenv("OPEN_WEATHER_MAP_URL", server.URL)
			fakeAPI, _ := NewOpenWeatherAPI(client)
			l := NewLocationEndpoint(test.db, fakeAPI)

			var bodyString string
			if len(test.countryCode) > 0 {
				bodyString = fmt.Sprintf(`{"city_name": "%s", "country_code": "%s"}`, test.cityName, test.countryCode)
			} else if len(test.cityName) > 0 {
				bodyString = fmt.Sprintf(`{"city_name": "%s"}`, test.cityName)
			} else {
				bodyString = `{"invalid": "Warsaw"}`
			}

			bodyReader := strings.NewReader(bodyString)
			httpRequest, _ := http.NewRequest("POST", "/locations", bodyReader)
			if len(test.contentType) > 0 {
				httpRequest.Header.Set("Content-Type", test.contentType)
			} else {
				httpRequest.Header.Set("Content-Type", "application/json")
			}
			request := restful.NewRequest(httpRequest)

			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)
			params := request.PathParameters()
			params["city_name"] = test.cityName

			// Act
			l.createLocation(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
				return
			}

			assert.Nil(t, response.Error())
			res := response.ResponseWriter.(*httptest.ResponseRecorder)
			if assert.NotNil(t, res) {
				l := Location{}
				err := json.Unmarshal(res.Body.Bytes(), &l)
				assert.Nil(t, err)
				assert.Equal(t, test.db.locations[0], l)
			}

		})
	}
}

func TestLocationEnpoint(t *testing.T) {
	t.Run("Check location endpoint settings", func(t *testing.T) {
		// Arrange
		externalAPI, _ := NewOpenWeatherAPI(nil)
		l := NewLocationEndpoint(nil, externalAPI)

		// Act
		ws := l.Endpoint()

		// Assert
		require.NotNil(t, ws)
		assert.Equal(t, "/locations", ws.RootPath())
		routes := ws.Routes()
		assert.Len(t, routes, 4)
	})
}
