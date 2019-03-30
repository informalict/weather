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
		externalAPI := NewOpenWeatherAPI(nil)
		l := NewStatisticsEndpoint(nil, externalAPI)

		// Act
		ws := l.Endpoint()

		// Assert
		require.NotNil(t, ws)
		assert.Equal(t, "/weather", ws.RootPath())
		routes := ws.Routes()
		assert.Len(t, routes, 1)
	})
}

func TestGetWeather(t *testing.T) {
	// Arrange
	tests := []struct {
		name                string
		expectedError       error
		LocationId          string
		db                  fakeDatabase
		HTTPStatus          int
		externalAPIResponse string
		contentType         string
	}{
		{
			name:          "Bad request",
			LocationId:    "abc",
			expectedError: fmt.Errorf("location_id must be an integer"),
			HTTPStatus:    http.StatusBadRequest,
			contentType:   "application/invalid",
		},
		{
			name:          "Location does not exist",
			LocationId:    "123",
			expectedError: fmt.Errorf("location '123' not found"),
			HTTPStatus:    http.StatusNotFound,
			db: fakeDatabase{
				err: DBNoRows,
			},
		},
		{
			name:          "Can not get location from database",
			LocationId:    "123",
			expectedError: fmt.Errorf("service is unavailable"),
			HTTPStatus:    http.StatusServiceUnavailable,
			db: fakeDatabase{
				err: errors.New("database error"),
			},
		},
		{
			name:                "Invalid format data from open map weather service",
			LocationId:          "123",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusBadGateway,
			externalAPIResponse: `{invalid json}`,
		},
		{
			name:                "Can not save statistics",
			LocationId:          "123",
			expectedError:       fmt.Errorf("service is unavailable"),
			HTTPStatus:          http.StatusServiceUnavailable,
			externalAPIResponse: `{ "weather": [ { "main": "Cloudy" } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
			db: fakeDatabase{
				errStat: errors.New("database error"),
			},
		},
		{
			name:                "Statistics has been saved",
			LocationId:          "123",
			HTTPStatus:          http.StatusCreated,
			externalAPIResponse: `{ "weather": [ { "main": "Cloudy" } ], "main": { "temp": 290.85, "temp_min": 288.71, "temp_max": 293.15 } }`,
			db: fakeDatabase{
				statistics: Statistic{
					Type:        "Cloudy",
					LocationId:  123,
					Temperature: 290.85,
					TempMin:     288.71,
					TempMax:     293.15,
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
				rw.WriteHeader(test.HTTPStatus)
				rw.Header()
				rw.Write([]byte(test.externalAPIResponse))
			}))
			defer server.Close()

			client := server.Client()
			client.Timeout = time.Second

			os.Setenv("OPEN_WEATHER_MAP_URL", server.URL)
			fakeAPI := NewOpenWeatherAPI(client)
			l := NewStatisticsEndpoint(test.db, fakeAPI)

			bodyString := fmt.Sprintf(`{"location_id": "%s"}`, test.LocationId)
			bodyReader := strings.NewReader(bodyString)
			httpRequest, _ := http.NewRequest("POST", "/weather", bodyReader)
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
			params["location_id"] = test.LocationId

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
				s := Statistic{}
				err := json.Unmarshal(res.Body.Bytes(), &s)
				assert.Nil(t, err)
				assert.Equal(t, test.db.statistics, s)
			}
		})
	}
}
