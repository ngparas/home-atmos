#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE USER appuser WITH password '${APPUSER_PASSWORD}';
    CREATE USER ingestuser WITH password '${INGESTUSER_PASSWORD}';
    CREATE DATABASE appdb;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d appdb <<-EOSQL
    CREATE TABLE readings (
        device_id VARCHAR NOT NULL,
        signal_name VARCHAR NOT NULL,
        signal_time TIMESTAMP WITH TIME ZONE NOT NULL,
        signal_value NUMERIC,
        PRIMARY KEY(device_id, signal_name, signal_time)
    );
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d appdb <<-EOSQL
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO appuser;
    GRANT SELECT, UPDATE, INSERT, DELETE ON ALL TABLES IN SCHEMA public TO ingestuser;
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ingestuser;
EOSQL



