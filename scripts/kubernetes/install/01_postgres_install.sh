#!/usr/bin/env bash

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"

source "$PROJECT_HOME/scripts/common/load_env.sh"

if [ ! -d "$PSQL_RESOURCES_DIR" ]; then
    echo "PostgreSQL resources directory does not exist: $PSQL_RESOURCES_DIR"
    exit 1
fi

# Create namespace for our PSQL databases (if it does not already exist)
kubectl get namespace | grep -q "^$PROJECT_NAMESPACE" || kubectl create namespace $PROJECT_NAMESPACE

# PostgreSql version 17.50 deployed by bitnami helm chart version 16.7.8

helm install my-postgresql \
    --set primary.extendedConfiguration="wal_level=logical" \
    --set global.postgresql.auth.postgresPassword="$PSQL_MASTER_PASSWORD" \
      bitnami/postgresql --namespace $PROJECT_NAMESPACE --version 16.7.8

# Wait for the postgresql pod to be available

while [[ $(kubectl -n $PROJECT_NAMESPACE get pods -l app.kubernetes.io/name=postgresql -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for postgresql pod to be available..." && sleep 30; done

echo "-----------------------"
echo "Postgresql is available"
echo "-----------------------"

# Change to the PostgreSQL resources directory
if ! cd "$PSQL_RESOURCES_DIR"; then
    echo "Failed to change directory to $PSQL_RESOURCES_DIR"
    exit 1
fi

# Ensure that the envsubst command is available
if ! command -v envsubst &> /dev/null; then
  echo "envsubst command not found. Please install gettext package."
  exit 1
fi

# Show environment variables that will be substituted
echo "Substituting environment variables in the template file:"
# env | grep -E 'PSQL_MASTER_PASSWORD|PSQL_SHOPPING_CARTS_DB|PSQL_SHOPPING_CARTS_ROLE|PSQL_SHOPPING_CARTS_PASSWORD|PSQL_ORDERS_DB|PSQL_ORDERS_ROLE|PSQL_ORDERS_PASSWORD|PSQL_CUSTOMERPROFILES_DB|PSQL_CUSTOMERPROFILES_ROLE|PSQL_CUSTOMERPROFILES_PASSWORD|PSQL_KEYCLOAK_DB|PSQL_KEYCLOAK_ROLE|PSQL_KEYCLOAK_PASSWORD'

# Loop through all .template files in the PostgreSQL resources directory
# and substitute the variables using envsubst
# This will create the SQL files needed to create the databases and roles
# Use the output files (in sorted order) to execute the SQL commands against the PostgreSQL database

find "$PSQL_RESOURCES_DIR" -type f -name '*.template' | sort | while read -r template; do
  output="${template%.template}"  # Remove .template extension for output file
  # Check if the output file already exists
  if [ -f "$output" ]; then
    echo "Output file $output already exists - deleting..."
    rm "$output"
  fi
  echo "Substituting variables in $template -> $output"
  envsubst < "$template" > "$output"

  # Decide which DB/user/password to use based on the filename
  case "$output" in
    *create_databases.sql)
      DB=postgres
      USER=postgres
      PGPASSWORD="$PSQL_MASTER_PASSWORD"
      ;;
    *customerprofiles_schema.sql)
      DB="$PSQL_CUSTOMER_PROFILES_DB"
      USER="$PSQL_CUSTOMER_PROFILES_ROLE"
      PGPASSWORD="$PSQL_CUSTOMER_PROFILES_PASSWORD"
      ;;
    *orders_schema.sql)
      DB="$PSQL_ORDERS_DB"
      USER="$PSQL_ORDERS_ROLE"
      PGPASSWORD="$PSQL_ORDERS_PASSWORD"
      ;;
    *shoppingcarts_schema.sql)
      DB="$PSQL_SHOPPING_CARTS_DB"
      USER="$PSQL_SHOPPING_CARTS_ROLE"
      PGPASSWORD="$PSQL_SHOPPING_CARTS_PASSWORD"
      ;;
    *)
      echo "Unknown SQL file type: $output, skipping execution."
      continue
      ;;
  esac

  echo "Running $output against database $DB as user $USER"
  kubectl run psql-client --rm -i --restart='Never' --namespace "$PROJECT_NAMESPACE" \
    --image docker.io/bitnami/postgresql:17.5.0-debian-12-r8 \
    --env="PGPASSWORD=$PGPASSWORD" \
    --command -- psql --host my-postgresql -U "$USER" -d "$DB" -p 5432 < "$output"

  if [ $? -ne 0 ]; then
    echo "Failed to execute $output"
    exit 1
  fi
  rm "$output"
done

echo
echo "All SQL files processed and executed successfully."
echo "PostgreSQL databases and roles created successfully in namespace '$PROJECT_NAMESPACE'."