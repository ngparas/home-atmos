#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE USER appuser WITH password '${APPUSER_PASSWORD}';
    CREATE USER ingestuser WITH password '${INGESTUSER_PASSWORD}';
    CREATE DATABASE appdb;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d appdb <<-EOSQL
    CREATE TABLE readings (
        idx SERIAL PRIMARY KEY,
        device_id VARCHAR NOT NULL,
        time_valid TIMESTAMP WITH TIME ZONE,
        signal_name VARCHAR,
        signal_value NUMERIC
    );
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d appdb <<-EOSQL
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO appuser;
    GRANT SELECT, UPDATE, INSERT, DELETE ON ALL TABLES IN SCHEMA public TO ingestuser;
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ingestuser;
EOSQL



