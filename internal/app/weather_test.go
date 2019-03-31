package app

import (
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
	tests := []struct {
		name                string
		expectedError       error
		LocationID          string
		db                  fakeDatabase
		HTTPStatus          int
		externalAPIResponse string
		externalAPITimeout  time.Duration
	}{
		{
			name:          "Bad request",
			LocationID:    "abc",
			expectedError: fmt.Errorf("location_id must be an integer"),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:          "Location does not exist",
			LocationID:    "123",
			expectedError: fmt.Errorf("location '123' not found"),
			HTTPStatus:    http.StatusNotFound,
			db: fakeDatabase{
				err: ErrDBNoRows,
			},
		},
		{
			name:          "Can not get location from database",
			LocationID:    "123",
			expectedError: fmt.Errorf("service is unavailable"),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				err: errors.New("database error"),
			},
		},
		{
			name:                "Invalid format data from open map weather service",
			LocationID:          "123",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusBadGateway,
			externalAPIResponse: `{invalid json}`,
		},
		{
			name:               "Open weather API timeout",
			LocationID:         "123",
			expectedError:      fmt.Errorf("service is unavailable"),
			HTTPStatus:         http.StatusGatewayTimeout,
			externalAPITimeout: time.Second * 2,
		},
		{
			name:                "Can not save statistics",
			LocationID:          "123",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusServiceUnavailable,
			externalAPIResponse: `{ "weather": [ { "main": "Cloudy" } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
			db: fakeDatabase{
				errSave: errors.New("database error"),
			},
		},
		{
			name:                "Statistics has been saved",
			LocationID:          "123",
			HTTPStatus:          http.StatusCreated,
			externalAPIResponse: `{ "weather": [ { "main": "Rain", "id": 501 } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
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
			expectedError: fmt.Errorf("location_id must be an integer"),
			HTTPStatus:    http.StatusBadRequest,
		},
		{
			name:          "Can not get statistics",
			LocationID:    "123",
			expectedError: fmt.Errorf("service is unavailable"),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				err: errors.New("database error"),
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
