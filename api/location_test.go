package api

import (
	"errors"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

//func (l *LocationMock) getDBLocation() error {
//	l.CityName = "sss"
//	return nil
//}
type dbMock struct {
	err error
}

func (d dbMock) getDBLocation(id int) (Location, error) {
	if d.err != nil {
		return Location{}, d.err //TODO nil instead of location
	}
	return Location{CityName: "sfddsa"}, nil
}

func (dbMock) getDBLocations() ([]Location, error) {
	return []Location{
		{
			CityName: "fdsafds",
		},
		{
			CityName: "gfdgfd",
		},
	}, nil
}

func (dbMock) saveDBLocation(location *Location) (*Location, error) {
	return &Location{}, nil
}

func (dbMock) deleteDBLocation(id int) error {
	return nil
}

func TestGetLocation(t *testing.T) {

	tests := []struct {
		name          string
		expectedError error
		locationId    string
		mock          dbMock
	}{
		{
			name:          "Invalid location_id",
			locationId:    "invalid",
			expectedError: errors.New("location_id must be an integer"),
		},
		{
			name:       "Invalid location_id",
			locationId: "123",
		},
		{
			name:       "No connection to database",
			locationId: "123",
			mock: dbMock{
				err: errors.New("can not connect to database"),
			},
			expectedError: errors.New("service is unavailable"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := NewLocation(test.mock)
			request := restful.NewRequest(nil)
			httpWriter := httptest.NewRecorder()
			response := restful.NewResponse(httpWriter)
			params := request.PathParameters()
			params["location_id"] = test.locationId

			//logger.Init("fsd")// TODO turn of logger

			l.getLocation(request, response)
			if test.expectedError != nil {
				assert.EqualError(t, response.Error(), test.expectedError.Error())
			} else {
				//TODO how to read status and body from response
			}
		})
	}

	//t.Run("Invalid location_id", func(t *testing.T) {
	//	l := Location{}
	//	request := restful.NewRequest(nil)
	//
	//	httpWriter := httptest.NewRecorder()
	//	response := restful.NewResponse(httpWriter)
	//	params := request.PathParameters()
	//	params["location_id"] = "abc"
	//
	//	l.getLocation(request, response)
	//	assert.EqualError(t, response.Error(), "location_id must be an integer")
	//})
	//
	//t.Run("Invalid location_id", func(t *testing.T) {
	//
	//	db := new(dbMock)
	//	l := NewLocation(db)
	//	request := restful.NewRequest(nil)
	//	httpWriter := httptest.NewRecorder()
	//	response := restful.NewResponse(httpWriter)
	//	params := request.PathParameters()
	//	params["location_id"] = "123"
	//
	//	l.getLocation(request, response)
	//	fmt.Println(response)
	//	params["location_id"] = "123"
	//	//assert.EqualError(t, response.(), "location_id must be an integer")
	//})

	//t.Run("TODO", func(t *testing.T) {
	//
	//	db := new(dbMock)
	//	l := NewLocation(db)
	//	request := restful.NewRequest(nil)
	//	httpWriter := httptest.NewRecorder()
	//	response := restful.NewResponse(httpWriter)
	//	params := request.PathParameters()
	//	params["location_id"] = "123"
	//
	//	l.getLocations(request, response)
	//	fmt.Println(response)
	//	params["location_id"] = "123"
	//	//assert.EqualError(t, response.(), "location_id must be an integer")
	//})

	//request := restful.Request{}
	//request.
	//request.PathParameters()
	//
	//l.getLocation(request, response)
	//assert.Equal()
}
