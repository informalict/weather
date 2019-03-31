package app

type fakeDatabase struct {
	err        error
	errStat    error
	locations  []Location
	weather    Weather
	statistics Statistics
}

func (f fakeDatabase) getDBLocation(id int) (Location, error) {
	if len(f.locations) > 0 {
		return f.locations[0], f.err
	}
	return Location{}, f.err
}

func (f fakeDatabase) getDBLocations() ([]Location, error) {
	return f.locations, f.err
}

func (f fakeDatabase) saveDBLocation(location Location) error {
	if len(f.locations) > 0 {
		location = f.locations[0]
	}
	return f.err
}

func (f fakeDatabase) deleteDBLocation(id int) error {
	return f.err
}

func (f fakeDatabase) saveDBWeather(s Weather) error {
	return f.errStat
}

func (f fakeDatabase) getStatistics(id int) (Statistics, error) {
	return f.statistics, f.err
}
