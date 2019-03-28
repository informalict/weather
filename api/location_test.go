package api

import (
	"fmt"
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
}

func (dbMock) getDBLocation(id int) (Location, error) {
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

func TestGetLocation(t *testing.T) {

	t.Run("Invalid location_id", func(t *testing.T) {
		l := Location{}
		request := restful.NewRequest(nil)

		httpWriter := httptest.NewRecorder()
		response := restful.NewResponse(httpWriter)
		params := request.PathParameters()
		params["location_id"] = "abc"

		l.getLocation(request, response)
		assert.EqualError(t, response.Error(), "location_id must be an integer")
	})

	t.Run("Invalid location_id", func(t *testing.T) {

		db := new(dbMock)
		l := NewLocation(db)
		request := restful.NewRequest(nil)
		httpWriter := httptest.NewRecorder()
		response := restful.NewResponse(httpWriter)
		params := request.PathParameters()
		params["location_id"] = "123"

		l.getLocation(request, response)
		fmt.Println(response)
		params["location_id"] = "123"
		//assert.EqualError(t, response.(), "location_id must be an integer")
	})

	t.Run("TODO", func(t *testing.T) {

		db := new(dbMock)
		l := NewLocation(db)
		request := restful.NewRequest(nil)
		httpWriter := httptest.NewRecorder()
		response := restful.NewResponse(httpWriter)
		params := request.PathParameters()
		params["location_id"] = "123"

		l.getLocations(request, response)
		fmt.Println(response)
		params["location_id"] = "123"
		//assert.EqualError(t, response.(), "location_id must be an integer")
	})

	//request := restful.Request{}
	//request.
	//request.PathParameters()
	//
	//l.getLocation(request, response)
	//assert.Equal()
}
