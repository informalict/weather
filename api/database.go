package api

import (
	"github.com/go-pg/pg"
	"os"
)

type DBConfig struct {
	database *pg.Options
}

type DatabaseLocation interface {
	getDBLocation(id int) (Location, error)
}

func NewDB() *DBConfig {
	return &DBConfig{
		&pg.Options{
			User:     os.Getenv("DB_USER"),     //"postgres",       //todo env
			Database: os.Getenv("DB_DATABASE"), //"weather",            //todo env
			Password: os.Getenv("DB_PASSWORD"), //"postgres",               //todo env
			Addr:     os.Getenv("DB_ADDRESS"),  //"localhost:5432",         //todo env
		},
	}
}

func (d *DBConfig) getDBLocation(id int) (Location, error) {
	a := pg.Connect(d.database)
	defer a.Close()

	location := Location{}
	err := a.Model(&location).Where("location_id = ?", id).Select()
	return location, err
}
