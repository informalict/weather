package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/google/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type fakeLocationDB struct {
	err       error
	locations []Location
}

func (f fakeLocationDB) getDBLocation(id int) (Location, error) {
	if len(f.locations) > 0 {
		return f.locations[0], f.err
	}
	return Location{}, f.err
}

func (f fakeLocationDB) getDBLocations() ([]Location, error) {
	return f.locations, f.err
}

func (f fakeLocationDB) saveDBLocation(location Location) error {
	if len(f.locations) > 0 {
		location = f.locations[0]
	}
	return f.err
}

func (f fakeLocationDB) deleteDBLocation(id int) error {
	return f.err
}

func TestMain(m *testing.M) {
	//TODO how to turn of flogger
	lf, err := os.OpenFile("./testing.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", lf)
	}
	defer lf.Close()

	log := logger.Init("Testing", true, true, lf)
	defer log.Close()

	code := m.Run()
	os.Exit(code)
}

func TestGetLocation(t *testing.T) {
	// Arrange
	tests := []struct {
		name          string
		expectedError error
		locationId    string
		db            fakeLocationDB
		HTTPStatus    int
	}{
		{
			name:          "Invalid location_id",
			locationId:    "invalid",
			expectedError: errors.New("location_id must be an integer"),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:       "No connection to database",
			locationId: "123",
			db: fakeLocationDB{
				err: errors.New("can not connect to database"),
			},
			expectedError: errors.New("service is unavailable"),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "Location not found in database",
			locationId: "462356",
			db: fakeLocationDB{
				err: DBNoRows,
			},
			HTTPStatus:    http.StatusNotFound,
			expectedError: errors.New("location '462356' not found"),
		},
		{
			name:       "Get valid location",
			locationId: "123",
			db: fakeLocationDB{
				locations: []Location{
					{
						LocationId:  123,
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
			params["location_id"] = test.locationId

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
		locationId    string
		db            fakeLocationDB
		HTTPStatus    int
	}{
		{
			name:          "Invalid location_id",
			locationId:    "invalid",
			expectedError: errors.New("location_id must be an integer"),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:       "No entry in database",
			locationId: "123",
			db: fakeLocationDB{
				err: DBNoRows,
			},
			expectedError: errors.New("location '123' does not exist"),
			HTTPStatus:    http.StatusNotFound,
		},
		{
			name:       "Database error",
			locationId: "123",
			db: fakeLocationDB{
				err: errors.New("database error"),
			},
			expectedError: errors.New("can not delete location '123'"),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "Location has been deleted",
			locationId: "123",
			HTTPStatus: http.StatusOK,
			db: fakeLocationDB{
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
			params["location_id"] = test.locationId

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
		db            fakeLocationDB
		HTTPStatus    int
	}{
		{
			name: "Database error",
			db: fakeLocationDB{
				err: errors.New("database error"),
			},
			expectedError: errors.New("service is unavailable"),
			HTTPStatus:    http.StatusServiceUnavailable,
		},
		{
			name:       "The list with two locations",
			HTTPStatus: http.StatusOK,
			db: fakeLocationDB{
				locations: []Location{
					{
						LocationId:  123,
						CityName:    "Warsaw",
						CountryCode: "PL",
						Longitude:   21.01,
						Latitude:    52.23,
					},
					{
						LocationId:  2643743,
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
	tests := []struct {
		name                string
		expectedError       error
		cityName            string
		countryCode         string
		db                  fakeLocationDB
		HTTPStatus          int
		externalAPIResponse string
		externalAPITimeout  time.Duration
		contentType         string
	}{
		{
			name:               "Bad request",
			expectedError:      fmt.Errorf("invalid data input"),
			HTTPStatus:         http.StatusBadRequest,
			externalAPITimeout: time.Second * 2,
			contentType:        "application/invalid",
		},
		{
			name:               "Open weather API timeout",
			cityName:           "Warsaw",
			expectedError:      fmt.Errorf("service is unavailable"),
			HTTPStatus:         http.StatusGatewayTimeout,
			externalAPITimeout: time.Second * 2,
		},
		{
			name:                "Invalid format data from open map weather service",
			cityName:            "Warsaw",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusBadGateway,
			externalAPIResponse: `{invalid json}`,
		},
		{
			name:                "Location can not be saved in database",
			cityName:            "Warsaw",
			countryCode:         "PL",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusServiceUnavailable,
			externalAPIResponse: `{"city_name": "Warsaw", "country_code": "PL"}`,
			db: fakeLocationDB{
				err: fmt.Errorf("cannot connect to database"),
			},
		},
		{
			name:                "Location has been saved",
			cityName:            "Warsaw",
			HTTPStatus:          http.StatusCreated,
			externalAPIResponse: `{ "id": 756135, "name": "Warsaw", "sys": { "country": "PL" } }`,
			db: fakeLocationDB{
				locations: []Location{
					{
						LocationId:  756135,
						CityName:    "Warsaw",
						CountryCode: "PL",
					},
				},
			},
		},
	}

	urlOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
	defer func() {
		os.Setenv("OPEN_WEATHER_MAP_URL", urlOriginal)
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if test.externalAPITimeout > 0 {
					time.Sleep(test.externalAPITimeout) //timeout simulation
				}
				rw.WriteHeader(test.HTTPStatus)
				rw.Header()
				rw.Write([]byte(test.externalAPIResponse))
			}))
			defer server.Close()

			client := server.Client()
			client.Timeout = time.Second

			os.Setenv("OPEN_WEATHER_MAP_URL", server.URL)
			fakeAPI := NewOpenWeatherAPI(client)
			l := NewLocationEndpoint(test.db, fakeAPI)

			var bodyString string
			if len(test.countryCode) > 0 {
				bodyString = fmt.Sprintf(`{"city_name": "%s", "country_code": "%s"}`, test.cityName, test.countryCode)
			} else {
				bodyString = fmt.Sprintf(`{"city_name": "%s"}`, test.cityName)
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

func TestWebService(t *testing.T) {
	t.Run("Check endpoint settings", func(t *testing.T) {
		// Arrange
		externalAPI := NewOpenWeatherAPI(nil)
		l := NewLocationEndpoint(nil, externalAPI)

		// Act
		ws := l.WebService()

		// Assert
		require.NotNil(t, ws)
		assert.Equal(t, "/locations", ws.RootPath())
		routes := ws.Routes()
		assert.Len(t, routes, 4)
	})
}