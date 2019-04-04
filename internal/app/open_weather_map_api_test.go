package app

import (
	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestBuildURI(t *testing.T) {
	URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
	tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

	defer func() {
		os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
	}()

	os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
	os.Setenv("OPEN_WEATHER_MAP_TOKEN", "test_token")

	t.Run("check build url", func(t *testing.T) {
		// Arrange
		o, _ := NewOpenWeatherAPI(nil)

		// Act
		uri := o.buildURI("enpoint", map[string]string{
			"key1": "value1",
		})

		// Assert
		assert.Equal(t, "http://test_url/enpoint?appid=test_token&key1=value1", uri)
	})
}

func TestNewOpenWeatherAPI(t *testing.T) {
	t.Run("Invalid configuration for open weather map client", func(t *testing.T) {
		// Arrange
		URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
		tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

		defer func() {
			os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
			os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
		}()

		os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", "")
		o, err := NewOpenWeatherAPI(nil)

		assert.Nil(t, o)
		assert.NotNil(t, err)
	})
}

func TestParseErrorResponse(t *testing.T) {
	t.Run("Parse valid error response", func(t *testing.T) {
		// Arrange
		URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
		tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

		defer func() {
			os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
			os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
		}()

		os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", "token")
		o, _ := NewOpenWeatherAPI(nil)
		require.NotNil(t, o)
		test := &OpenMapWeatherError{
			Message: "unknown error from open weather map service",
		}

		response := httptest.NewRecorder()
		response.WriteHeader(http.StatusOK)
		b, err := json.Marshal(test)
		require.Nil(t, err)
		response.Write(b)

		// Act
		err = o.parseErrorResponse(response.Result())

		// Assert
		require.NotNil(t, err)
		assert.Equal(t, test.Message, err.Error())
	})

	t.Run("Parse invalid error response", func(t *testing.T) {
		// Arrange
		URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
		tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

		defer func() {
			os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
			os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
		}()

		os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", "token")

		o, _ := NewOpenWeatherAPI(nil)
		require.NotNil(t, o)
		response := httptest.NewRecorder()
		response.WriteHeader(http.StatusOK)
		response.WriteString("invalid json")

		// Act
		err := o.parseErrorResponse(response.Result())

		// Assert
		assert.NotNil(t, err)
	})
}

func TestParseResponse(t *testing.T) {
	t.Run("Parse valid response", func(t *testing.T) {
		// Arrange
		URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
		tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

		defer func() {
			os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
			os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
		}()

		os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", "token")
		o, _ := NewOpenWeatherAPI(nil)
		require.NotNil(t, o)
		test := &OpenMapWeather{
			Name: "City",
			ID:   123,
		}

		response := httptest.NewRecorder()
		response.WriteHeader(http.StatusOK)
		b, err := json.Marshal(test)
		require.Nil(t, err)
		response.Write(b)

		// Act
		result, err := o.parseResponse(response.Result())

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, test, result)
	})

	t.Run("Parse invalid response", func(t *testing.T) {
		// Arrange
		URLOriginal := os.Getenv("OPEN_WEATHER_MAP_URL")
		tokenOriginal := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

		defer func() {
			os.Setenv("OPEN_WEATHER_MAP_URL", URLOriginal)
			os.Setenv("OPEN_WEATHER_MAP_TOKEN", tokenOriginal)
		}()

		os.Setenv("OPEN_WEATHER_MAP_URL", "http://test_url")
		os.Setenv("OPEN_WEATHER_MAP_TOKEN", "token")

		o, _ := NewOpenWeatherAPI(nil)
		require.NotNil(t, o)
		response := httptest.NewRecorder()
		response.WriteHeader(http.StatusOK)
		response.WriteString("invalid json")

		// Act
		result, err := o.parseResponse(response.Result())

		// Assert
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

}
