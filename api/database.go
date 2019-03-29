package api

import (
	"github.com/go-pg/pg"
	"os"
)

/*
//TODO create schema
CREATE TABLE locations (location_id INTEGER PRIMARY KEY, city_name VARCHAR NOT NULL,
country_code CHAR(10) NOT NULL, latitude numeric(6,2), longitude numeric(6,2), UNIQUE(city_name, country_code));
*/
type LocationDBOperations interface {
	getDBLocation(int) (Location, error)
	getDBLocations() ([]Location, error)
	saveDBLocation(*Location) (*Location, error)
	deleteDBLocation(int) error
}

type LocationDB struct {
	config *pg.Options
}

func NewDB() *LocationDB {
	return &LocationDB{
		&pg.Options{
			User:     os.Getenv("DB_USER"),
			Database: os.Getenv("DB_DATABASE"),
			Password: os.Getenv("DB_PASSWORD"),
			Addr:     os.Getenv("DB_ADDRESS"),
		},
	}
}

func (d *LocationDB) getDBLocation(id int) (Location, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	location := Location{}
	err := db.Model(&location).Where("location_id = ?", id).Select()
	return location, err
}

func (d *LocationDB) getDBLocations() ([]Location, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	var locations []Location
	err := db.Model(&locations).Order("country_code ASC", "city_name ASC").Select()
	return locations, err
}

func (d *LocationDB) saveDBLocation(location *Location) (*Location, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	if err := db.Insert(location); err != nil {
		return nil, err
	}
	return location, nil
}

func (d *LocationDB) deleteDBLocation(id int) error {
	db := pg.Connect(d.config)
	defer db.Close()

	location := Location{LocationId: id}
	_, err := db.Model(&location).Where("location_id = ?location_id").Delete()
	return err
}
