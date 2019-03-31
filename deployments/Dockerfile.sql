FROM postgres
ENV POSTGRES_DB weather
COPY configs/database.sql /docker-entrypoint-initdb.d/
