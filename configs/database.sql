CREATE TABLE locations (
location_id INTEGER PRIMARY KEY,
city_name VARCHAR NOT NULL,
country_code CHAR(4) NOT NULL,
latitude numeric(6,2),
longitude numeric(6,2),
UNIQUE(city_name, country_code)
);

CREATE TABLE weather(
id SERIAL PRIMARY KEY,
location_id INTEGER REFERENCES locations(location_id) ON DELETE CASCADE,
temperature numeric(6,2),
temp_min numeric(6,2),
temp_max numeric(6,2),
date DATE NOT NULL default CURRENT_DATE
);

CREATE INDEX weather_location ON weather(location_id);

CREATE TABLE conditions(
statistic_id INTEGER REFERENCES weather(id) ON DELETE CASCADE,
type INTEGER NOT NULL,
PRIMARY KEY(statistic_id, type)
);