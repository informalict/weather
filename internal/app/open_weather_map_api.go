package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

// Description stores open weather map internal data
type Description struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// OpenMapWeather stores open weather map internal data
type OpenMapWeather struct {
	Coord struct {
		Latitude  float32 `json:"lat"`
		Longitude float32 `json:"lon"`
	} `json:"coord"`
	Description []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp     float32 `json:"temp"`
		Pressure int     `json:"pressure"`
		Humidity int     `json:"humidity"`
		TempMin  float32 `json:"temp_min"`
		TempMax  float32 `json:"temp_max"`
	} `json:"main"`
	Visibility int `json:"int"`
	Wind       struct {
		Speed float32 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int `json:"dt"`
	Sys struct {
		Type    int     `json:"type"`
		ID      int     `json:"id"`
		Message float32 `json:"message"`
		Country string  `json:"country"`
		Sunrise int     `json:"sunrise"`
		Sunset  int     `json:"sunset"`
	} `json:"sys"`
	ID   int    `json:"id"`
	Name string `json:"name"`
	//Cpd  int    `json:"cod"`
}

// OpenWeatherAPI is a client for open weather map service
type OpenWeatherAPI struct {
	client  *http.Client
	baseURL string
	token   string
}

// OpenMapWeatherError stores cause of error from open weather map service
type OpenMapWeatherError struct {
	Message string `json:"message"`
}

// NewOpenWeatherAPI returns new client to open weather map service
func NewOpenWeatherAPI(client *http.Client) (*OpenWeatherAPI, error) {
	baseURL := os.Getenv("OPEN_WEATHER_MAP_URL")
	token := os.Getenv("OPEN_WEATHER_MAP_TOKEN")

	if len(baseURL) == 0 || len(token) == 0 {
		return nil, errors.New("configuration for open weather map client is not provided")
	}

	return &OpenWeatherAPI{
		client:  client,
		baseURL: baseURL,
		token:   token,
	}, nil
}

func (o *OpenWeatherAPI) buildURI(endpoint string, params map[string]string) string {
	uri := fmt.Sprintf("%s/%s?appid=%s", o.baseURL, endpoint, o.token)
	for k, v := range params {
		uri += fmt.Sprintf("&%s=%s", k, v)
	}
	return uri
}

func (o *OpenWeatherAPI) parseErrorResponse(response *http.Response) error {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	message := &OpenMapWeatherError{}
	if err = json.Unmarshal(body, message); err != nil {
		return fmt.Errorf("%s body=(%s)", err.Error(), body)
	}
	return errors.New(message.Message)
}

func (o *OpenWeatherAPI) parseResponse(response *http.Response) (*OpenMapWeather, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	weather := &OpenMapWeather{}
	if err = json.Unmarshal(body, weather); err != nil {
		return nil, fmt.Errorf("%s body=(%s)", err.Error(), body)
	}
	return weather, nil
}

func (o *OpenWeatherAPI) getWeather(params map[string]string) (*OpenMapWeather, int, error) {
	uri := o.buildURI("weather", params)
	resp, err := o.client.Get(uri)
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, http.StatusGatewayTimeout, err
		}
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, http.StatusNotFound, o.parseErrorResponse(resp)
		}
		return nil, http.StatusBadGateway, o.parseErrorResponse(resp)
	}

	response, err := o.parseResponse(resp)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	return response, http.StatusOK, nil

}
