#!/usr/bin/env bash

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"

source "$PROJECT_HOME/scripts/common/load_env.sh"

if [ ! -d "$PSQL_MODELS_DIR" ]; then
    echo "PostgreSQL models directory does not exist: $PSQL_MODELS_DIR"
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

# Change to the models directory
if ! cd "$PSQL_MODELS_DIR"; then
    echo "Failed to change directory to $PSQL_MODELS_DIR"
    exit 1
fi

# Ensure that the envsubst command is available
if ! command -v envsubst &> /dev/null; then
    echo "envsubst command not found. Please install gettext package."
    exit 1
fi

# Loop through all <domain>_db.sql files under the models directory to create the databases and roles
# and then create the schemas from the <domain>_schema.sql files that correspond to each template
echo
echo "--------------------------------------------"
echo "Creating databases and roles from db_templates in $PSQL_MODELS_DIR"
echo

find "$PSQL_MODELS_DIR" -type f -name '*_db.sql' | sort | while read -r template; do
  
    outputdb="${template%_db.sql}.db"
    if [ -f "$outputdb" ]; then
        echo "Output file $outputdb already exists - deleting..."
        rm "$outputdb"
    fi

    echo
    echo "Processing database template: $template"
    envsubst < "$template" > "$outputdb"
    if [ $? -ne 0 ]; then
        echo "Failed to substitute environment variables in $template"
        continue
    fi
    
    # Check if the output file is empty
    if [ ! -s "$outputdb" ]; then
        echo "Output file $outputdb is empty after substitution. Skipping."
        continue
    fi
    
    # Extract DB, USER, PGPASSWORD from the output file's metadata comments
    DB=$(grep -m1 '^-- DB:' "$outputdb" | sed 's/^-- DB:[[:space:]]*//')
    USER=$(grep -m1 '^-- USER:' "$outputdb" | sed 's/^-- USER:[[:space:]]*//')
    PGPASSWORD=$(grep -m1 '^-- PGPASSWORD:' "$outputdb" | sed 's/^-- PGPASSWORD:[[:space:]]*//')

    if [[ -z "$DB" || -z "$USER" || -z "$PGPASSWORD" ]]; then
        echo "Missing DB, USER, or PGPASSWORD metadata in $outputdb. Skipping."
        continue
    fi

    echo "Creating database..."
    kubectl run psql-client --rm -i --restart='Never' --namespace "$PROJECT_NAMESPACE" \
        --image docker.io/bitnami/postgresql:17.5.0-debian-12-r8 \
        --env="PGPASSWORD=$PGPASSWORD" \
        --command -- psql --host my-postgresql -U "$USER" -d "$DB" -p 5432 < "$outputdb"

    if [ $? -ne 0 ]; then
        echo "Failed to execute $outputdb"
        exit 1
    fi
    rm "$outputdb"
    echo

    # Now create the schema for this database using the corresponding <domain>_schema.sql file
    # The schema file should be named <domain>_schema.sql and located in the same directory as the template
    
    sqlfile="${template%_db.sql}_schema.sql"
    if [ ! -f "$sqlfile" ]; then
        echo "Schema file $sqlfile NOT found for template $template"
        exit 1
    fi

    outputsql="${sqlfile%.sql}_substituted.sql"
    if [ -f "$outputsql" ]; then
        echo "Output file $outputsql already exists - deleting..."
        rm "$outputsql"
    fi
    envsubst < "$sqlfile" > "$outputsql"
    
    # Extract DB, USER, PGPASSWORD from the sqlfile's metadata comments
    DB=$(grep -m1 '^-- DB:' "$outputsql" | sed 's/^-- DB:[[:space:]]*//')
    USER=$(grep -m1 '^-- USER:' "$outputsql" | sed 's/^-- USER:[[:space:]]*//')
    PGPASSWORD=$(grep -m1 '^-- PGPASSWORD:' "$outputsql" | sed 's/^-- PGPASSWORD:[[:space:]]*//')

    if [[ -z "$DB" || -z "$USER" || -z "$PGPASSWORD" ]]; then
        echo "Missing DB, USER, or PGPASSWORD metadata in $outputsql. Skipping."
        continue
    fi

    echo "Creating schema for database $DB as user $USER"
    kubectl run psql-client --rm -i --restart='Never' --namespace "$PROJECT_NAMESPACE" \
        --image docker.io/bitnami/postgresql:17.5.0-debian-12-r8 \
        --env="PGPASSWORD=$PGPASSWORD" \
        --command -- psql --host my-postgresql -U "$USER" -d "$DB" -p 5432 < "$outputsql"

    if [ $? -ne 0 ]; then
        echo "Failed to execute $outputsql"
        exit 1
    fi

    rm "$outputsql"
    echo

done

echo
echo "---------------"
echo "Script complete"
echo