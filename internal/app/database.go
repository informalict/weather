package app

import (
	"database/sql"
	"fmt"
	"github.com/go-pg/pg"
	"os"
)

type databaseWeatherProvider interface {
	getLocation(int) (Location, error)
	getLocations() ([]Location, error)
	saveLocation(Location) error
	deleteLocation(int) error
	saveWeather(Weather) error
	getStatistics(id int) (Statistics, error)
}

// Database is config struct for postgres connection
type Database struct {
	config *pg.Options
}

// NewDB creates new config for database connection
func NewDB() (db *Database, err error) {
	user := os.Getenv("DB_USER")
	database := os.Getenv("DB_DATABASE")
	password := os.Getenv("DB_PASSWORD")
	address := os.Getenv("DB_ADDRESS")

	if len(user) == 0 || len(database) == 0 || len(address) == 0 {
		err = fmt.Errorf("database configuration is not provided, user=(%s), address=(%s), databases=(%s)",
			user, address, database)
		return
	}

	return &Database{
		&pg.Options{
			User:     user,
			Database: database,
			Password: password,
			Addr:     address,
		},
	}, err
}

func (d *Database) getLocation(id int) (location Location, err error) {
	db := pg.Connect(d.config)
	defer db.Close()

	err = db.Model(&location).Where("location_id = ?", id).Select()
	if err == pg.ErrNoRows {
		err = sql.ErrNoRows
	}
	return
}

func (d *Database) getLocations() (locations []Location, err error) {
	db := pg.Connect(d.config)
	defer db.Close()

	err = db.Model(&locations).Order("country_code ASC", "city_name ASC").Select()
	return
}

func (d *Database) saveLocation(location Location) error {
	db := pg.Connect(d.config)
	defer db.Close()

	err := db.Insert(&location)
	return err
}

func (d *Database) deleteLocation(id int) error {
	db := pg.Connect(d.config)
	defer db.Close()

	location := Location{LocationID: id}
	v, err := db.Model(&location).Where("location_id = ?", id).Delete()
	if err == nil && v.RowsAffected() == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (d *Database) saveWeather(s Weather) error {
	db := pg.Connect(d.config)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err = tx.Insert(&s); err == nil {
		// Unfortunately there is no possibility to write record with relations
		for k := range s.Conditions {
			s.Conditions[k].StatisticID = s.ID
		}
		err = tx.Insert(&s.Conditions)
	}

	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func (d *Database) getStatistics(id int) (Statistics, error) {
	db := pg.Connect(d.config)
	defer db.Close()

	var err error
	s := Statistics{}

	// get the number of queries
	s.Count, err = db.Model(&Weather{}).Where("location_id = ?", id).Count()
	if err != nil {
		return s, err
	}

	// get min, max and average temperature for each month
	err = db.Model(&s.MonthTemperature).
		ColumnExpr("avg(temperature)").
		ColumnExpr("min(temp_min)").
		ColumnExpr("max(temp_max)").
		ColumnExpr("to_char(date, 'YYYY-MM') as month").
		Where("location_id = ?", id).Group("month").
		Select()
	if err != nil {
		return s, err
	}

	// get type of the weather and occurrence for each day for that type
	st, err := db.Prepare("SELECT date,type FROM weather AS w LEFT JOIN conditions AS c ON w.id=c.statistic_id " +
		"GROUP BY date,type ORDER BY date,type")
	if err != nil {
		return s, err
	}

	var lk []DailyConditionStatistics
	_, err = st.Query(&lk)
	if err != nil {
		return s, err
	}

	s.DailyCondition = make(map[string][]string)
	for _, v := range lk {
		s.DailyCondition[v.Date] = append(s.DailyCondition[v.Date], v.Type)
	}
	return s, err
}
