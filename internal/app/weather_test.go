package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStatisticsEndpoint(t *testing.T) {
	t.Run("Check statistics endpoint settings", func(t *testing.T) {
		// Arrange
		externalAPI, _ := NewOpenWeatherAPI(nil)
		l := NewWeatherEndpoint(nil, externalAPI)

		// Act
		ws := l.Endpoint()

		// Assert
		require.NotNil(t, ws)
		assert.Equal(t, "/weather", ws.RootPath())
		routes := ws.Routes()
		assert.Len(t, routes, 2)
	})
}

func TestGetWeather(t *testing.T) {
	// Arrange
	type ExternalAPI struct {
		response   string
		HTTPStatus int
		Timeout    time.Duration
	}

	tests := []struct {
		name          string
		expectedError error
		LocationID    string
		db            fakeDatabase
		HTTPStatus    int
		externalAPI   ExternalAPI
	}{
		{
			name:          "Bad request",
			LocationID:    "abc",
			expectedError: fmt.Errorf(locationInvalidID),
			HTTPStatus:    http.StatusBadRequest,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Location does not exist",
			LocationID:    "123",
			expectedError: fmt.Errorf("location '123' not found"),
			HTTPStatus:    http.StatusNotFound,
			db: fakeDatabase{
				err: sql.ErrNoRows,
			},
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Can not get location from database",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				err: errors.New("database error"),
			},
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
			},
		},
		{
			name:          "Invalid format data from open map weather service",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusBadGateway,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
				response:   `{invalid json}`,
			},
		},
		{
			name:          "Location does not exist in open weather map service",
			LocationID:    "1234567890",
			expectedError: fmt.Errorf("location '1234567890' not found"),
			HTTPStatus:    http.StatusNotFound,
			externalAPI: ExternalAPI{
				response:   `{ "cod": "404", "message": "city not found" }`,
				HTTPStatus: http.StatusNotFound,
			},
		},
		{
			name:          "Unknown error from open weather map service",
			LocationID:    "1234567890",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusBadGateway,
			externalAPI: ExternalAPI{
				response:   `{ "cod": "500", "message": "unknown" }`,
				HTTPStatus: http.StatusInternalServerError,
			},
		},
		{
			name:          "Open weather API timeout",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusGatewayTimeout,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
				Timeout:    time.Second * 2,
			},
		},
		{
			name:          "Can not save statistics",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
				response:   `{ "weather": [ { "main": "Cloudy" } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
			},
			db: fakeDatabase{
				errSave: errors.New("database error"),
			},
		},
		{
			name:       "Statistics has been saved",
			LocationID: "123",
			HTTPStatus: http.StatusOK,
			externalAPI: ExternalAPI{
				HTTPStatus: http.StatusOK,
				response:   `{ "weather": [ { "main": "Rain", "id": 501 } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
			},
			db: fakeDatabase{
				weather: Weather{
					Conditions: []Condition{
						{
							Type: "Rain",
						},
					},
					LocationID:  123,
					Temperature: 290.85,
					TempMin:     288.71,
					TempMax:     293.15,
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
			l := NewWeatherEndpoint(test.db, fakeAPI)

			bodyString := fmt.Sprintf(`{"location_id": "%s"}`, test.LocationID)
			bodyReader := strings.NewReader(bodyString)
			httpRequest, _ := http.NewRequest("POST", "/weather", bodyReader)
			httpRequest.Header.Set("Content-Type", "application/json")
			request := restful.NewRequest(httpRequest)

			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)
			params := request.PathParameters()
			params["location_id"] = test.LocationID

			// Act
			l.getWeather(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
				return
			}

			assert.Nil(t, response.Error())
			res := response.ResponseWriter.(*httptest.ResponseRecorder)
			if assert.NotNil(t, res) {
				weather := Weather{}
				err := json.Unmarshal(res.Body.Bytes(), &weather)
				assert.Nil(t, err)
				assert.Equal(t, test.db.weather, weather)
			}
		})
	}
}

func TestGetStatistics(t *testing.T) {
	// Arrange
	tests := []struct {
		name          string
		expectedError error
		LocationID    string
		db            fakeDatabase
		HTTPStatus    int
	}{
		{
			name:          "Bad request",
			LocationID:    "abc",
			expectedError: fmt.Errorf(locationInvalidID),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:          "Location does not exist",
			LocationID:    "123",
			expectedError: fmt.Errorf("location '123' does not exist"),
			HTTPStatus:    http.StatusNotFound,
			db: fakeDatabase{
				err: sql.ErrNoRows,
			},
		},
		{
			name:          "Can not get location",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				err: errors.New("get location database error"),
			},
		},
		{
			name:          "Can not get statistics",
			LocationID:    "123",
			expectedError: fmt.Errorf(serviceIsUnavailable),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				errStat: errors.New("get statistics database error"),
			},
		},
		{
			name:       "Statistics have been returned",
			LocationID: "123",
			HTTPStatus: http.StatusOK,
			db: fakeDatabase{
				statistics: Statistics{
					Count: 100,
					MonthTemperature: []MonthTemperatureStatistics{
						{
							Min:   299.15,
							Max:   302.59,
							Avg:   291,
							Month: "2018-03",
						},
						{
							Min:   279.15,
							Max:   282.59,
							Avg:   281,
							Month: "2019-03",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			w := NewWeatherEndpoint(test.db, nil)
			request := restful.NewRequest(nil)
			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			response.SetRequestAccepts(restful.MIME_JSON)
			params := request.PathParameters()
			params["location_id"] = test.LocationID

			w.getStatistics(request, response)

			// Assert
			assert.Equal(t, test.HTTPStatus, response.StatusCode())
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
				return
			}

			assert.Nil(t, response.Error())
			res := response.ResponseWriter.(*httptest.ResponseRecorder)
			if assert.NotNil(t, res) {
				s := Statistics{}
				err := json.Unmarshal(res.Body.Bytes(), &s)
				assert.Nil(t, err)
				assert.Equal(t, test.db.statistics, s)
			}
		})
	}
}
