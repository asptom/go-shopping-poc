-- This template is used to create the postgresql database and user for keycloak

-- The variables are expected to be set in a .ENV.* file and then replaced in this template using envsubst
-- in the installation shell script

-- The following lines will be used to set the DB, USER, and PGPASSWORD used to run the psqlclient
-- DB: $PSQL_MASTER_DB
-- USER: $PSQL_MASTER_USER
-- PGPASSWORD: $PSQL_MASTER_PASSWORD

DROP ROLE IF EXISTS $PSQL_KEYCLOAK_ROLE;
DROP DATABASE IF EXISTS $PSQL_KEYCLOAK_DB;

CREATE ROLE $PSQL_KEYCLOAK_ROLE WITH LOGIN PASSWORD '$PSQL_KEYCLOAK_PASSWORD';
CREATE DATABASE $PSQL_KEYCLOAK_DB OWNER $PSQL_KEYCLOAK_ROLE;