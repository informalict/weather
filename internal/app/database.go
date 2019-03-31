package app

import (
	"errors"
	"github.com/go-pg/pg"
	"os"
)

var (
	DBNoRows = errors.New("DATABASE_NO_ROWS")
)

type databaseProvider interface {
	getDBLocation(int) (Location, error)
	getDBLocations() ([]Location, error)
	saveDBLocation(Location) error
	deleteDBLocation(int) error
	saveDBWeather(Weather) error
	getStatistics(id int) (Statistics, error)
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

func (d *Database) saveDBWeather(s Weather) error {
	db := pg.Connect(d.config)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err = tx.Insert(&s); err == nil {
		// Unfortunately there is no possibility to write record with relations
		for k := range s.Conditions {
			s.Conditions[k].StatisticId = s.Id
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

	s := Statistics{}
	var err error
	s.Count, err = db.Model(&Weather{}).Where("location_id = ?", id).Count()
	if err != nil {
		return s, nil
	}

	err = db.Model(&s.MonthTemperature).
		ColumnExpr("avg(temperature)").
		ColumnExpr("min(temp_min)").
		ColumnExpr("max(temp_max)").
		ColumnExpr("to_char(date, 'YYYY-MM') as month").
		Where("location_id = ?", id).Group("month").
		Select()

	return s, err
	//err := db.Model(&s).
	//	ColumnExpr("statistics.date").
	//	ColumnExpr("conditions.type").
	//	Join("JOIN conditions ON statistics.id = conditions.statistic_id").
	//	Group("type", "date").
	//	First()

	//err := db.Model(&s).Column("statistic.id", "conditions.statistics_id").Select()

	//Join("inner join companies_customers cc on customer.id = cc.customer_id").Where("cc.company_id = ?", companyID).Select()

	//db.Model(&s).Join()
	//db.Prepare("select type,date from statistics AS s left join conditions AS c on s.id=c.statistic_id group by type,date"
	//select * from statistics where conditions @> '{"rain","cloudy"}';
}
