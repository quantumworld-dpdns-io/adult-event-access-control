FROM postgres:16-alpine

COPY migrations/*.sql /docker-entrypoint-initdb.d/
COPY seeds/ /seeds/

EXPOSE 5432
