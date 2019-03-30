package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

type OpenMapWeather struct {
	Coord struct {
		Latitude  float32 `json:"lat"`
		Longitude float32 `json:"lon"`
	} `json:"coord"`
	Wea []struct { //TODO change name
		Id          int    `json:"id"`
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
	Dt  string `json:"int"`
	Sys struct {
		Type    int     `json:"type"`
		Id      int     `json:"id"`
		Message float32 `json:"message"`
		Country string  `json:"country"`
		Sunrise int     `json:"sunrise"`
		Sunset  int     `json:"sunset"`
	} `json:sys`
	Id   int    `json:"id"`
	Name string `json:"name"`
	Cpd  int    `json:"cod"`
}

type OpenWeatherAPI struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewOpenWeatherAPI(client *http.Client) *OpenWeatherAPI {
	return &OpenWeatherAPI{
		client:  client,
		baseURL: os.Getenv("OPEN_WEATHER_MAP_URL"),   //TODO error if does not exist
		token:   os.Getenv("OPEN_WEATHER_MAP_TOKEN"), //TODO error if does not exist
	}
}

func (o *OpenWeatherAPI) buildURI(endpoint string, params map[string]string) string {
	uri := fmt.Sprintf("%s/%s?appid=%s", o.baseURL, endpoint, o.token)
	for k, v := range params {
		uri += fmt.Sprintf("&%s=%s", k, v)
	}
	return uri
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

func (o *OpenWeatherAPI) getWeather(params map[string]string) (*OpenMapWeather, error, int) {
	uri := o.buildURI("weather", params)
	resp, err := o.client.Get(uri)
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, err, http.StatusGatewayTimeout
		}
		return nil, err, http.StatusBadGateway
	}
	defer resp.Body.Close()

	response, err := o.parseResponse(resp)
	return response, err, http.StatusBadGateway
}
