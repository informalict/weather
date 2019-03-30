package app

import (
	"errors"
	"github.com/go-pg/pg"
	"os"
)

var (
	DBNoRows = errors.New("DATABASE_NO_ROWS")
)

/*
//TODO create schema
CREATE TABLE locations (
location_id INTEGER PRIMARY KEY,
city_name VARCHAR NOT NULL,
country_code CHAR(4) NOT NULL,//TODO 10?
latitude numeric(6,2),
longitude numeric(6,2),
UNIQUE(city_name, country_code)
);

CREATE TABLE statistics(
id SERIAL PRIMARY KEY,
location_id INTEGER REFERENCES locations(location_id),
temperature numeric(6,2),
temp_min numeric(6,2),
temp_max numeric(6,2),
type varchar
);

CREATE INDEX statistics_location ON statistics(location_id);
*/

type databaseProvider interface {
	getDBLocation(int) (Location, error)
	getDBLocations() ([]Location, error)
	saveDBLocation(Location) error
	deleteDBLocation(int) error
	saveDBStatistics(Statistic) error
}

type Database struct {
	config *pg.Options
}

func NewDB() *Database {
	return &Database{
		&pg.Options{
			User:     os.Getenv("DB_USER"),
			Database: os.Getenv("DB_DATABASE"),
			Password: os.Getenv("DB_PASSWORD"),
			Addr:     os.Getenv("DB_ADDRESS"),
		},
	}
}

func (d *Database) saveDBStatistics(s Statistic) error {
	db := pg.Connect(d.config)
	defer db.Close()

	return db.Insert(&s)
}

func (d *Database) getDBLocation(id int) (Location, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	location := Location{}
	err := db.Model(&location).Where("location_id = ?", id).Select()
	if err == pg.ErrNoRows {
		err = DBNoRows
	}
	return location, err
}

func (d *Database) getDBLocations() ([]Location, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	var locations []Location
	err := db.Model(&locations).Order("country_code ASC", "city_name ASC").Select()

	return locations, err
}

func (d *Database) saveDBLocation(location Location) error {
	db := pg.Connect(d.config)
	defer db.Close()

	return db.Insert(&location)
}

func (d *Database) deleteDBLocation(id int) error {
	db := pg.Connect(d.config)
	defer db.Close()

	location := Location{LocationId: id}
	v, err := db.Model(&location).Where("location_id = ?", id).Delete()
	if err == nil && v.RowsAffected() == 0 {
		return DBNoRows
	}
	return err
}
