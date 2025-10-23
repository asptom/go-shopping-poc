# This script loads environment variables from a .env.* file located in the project root directory.
# and is intended to be sourced in other scripts to ensure that the environment variables are available. 
#
# Usage: source $PROJECT_HOME/scripts/common/load_env.sh

# Provide safe defaults so scripts can be sourced under `set -u`
APP_ENV="${APP_ENV:-development}"
PROJECT_HOME="${PROJECT_HOME:-/Users/tom/Projects/Go/go-shopping-poc}"

if [ -z "$PROJECT_HOME" ]; then
  echo "PROJECT_HOME is not set. Please set it before sourcing this script."
  exit 1
fi

if [ -z "$APP_ENV" ]; then
  echo "APP_ENV is not set. Defaulting to 'development'."
  APP_ENV="development"
fi

if [[ -n "$APP_ENV" && -f "$PROJECT_HOME/.env.$APP_ENV" ]]; then
  echo "Loading environment variables from $PROJECT_HOME/.env.$APP_ENV"
  set -a
  source "$PROJECT_HOME/.env.$APP_ENV"
  set +a
fi