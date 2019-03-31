# Weather service
### Purpose
This service provides API that allows users to maintain favorite locations and generate for them weather statistics suc as:
* Minimum temperature for each month
* Maximum temperature for each month
* Average temperature for each month
* Number of statistics data
* Overall weather conditions aggregated by days 
### Configuration
###### Download and build images
* git clone git@github.com:mieczyslaw1980/weather.git
* cd weather
* make containers
###### Start application
OPEN_WEATHER_MAP_TOKEN=[YOUR_OPEN_WEATHER_MAP_API_TOKEN] docker-compose deployments/docker-compose.yml up
### Endpoints
1. Locations
* GET "/locations"
```$xslt
Get all user's locations
```
* GET "/locations/{id}"
```
Get one user's location
```
* DELETE "/locations/{id}"
```
Delete one user location
```
* POST "/locations"
```
Save new user's location by city name: 
   {"city_name": "London"}
Save new user's location by city name and country code:
   {"city_name": "London", "country_code": "GB"}
```
2. Weather
* GET "/weather/{id}"
```
Get current weather condition at a moment and save that for later statistis
```
* GET "/weather/{id}/statistics"
```
Calculate statistics for previous cumulated weather conditions
```
### Examples
1. Create new location by city name

curl -X POST -H "content-type: application/json" --data '{"city_name": "Warsaw"}' localhost:8080/locations
* HTTP Status 201
```$xslt
{
 "city_name": "Warsaw",
 "country_code": "PL",
 "location_id": 756135,
 "latitude": 52.23,
 "longitude": 21.01
}
```
2. Delete location
* curl -X DELETE  localhost:8080/locations/756135
```$xslt

```


TODO paste json here

### Tests
make test
