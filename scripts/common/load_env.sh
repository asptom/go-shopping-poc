# This script loads environment variables from a .env file located in the project root directory.
# and is intended to be sourced in other scripts to ensure that the environment variables are available. 
#
#Usage: source $PROJECT_HOME/scripts/common/load_env.sh

if [ -z "$PROJECT_HOME" ]; then
  echo "PROJECT_HOME is not set. Please set it before sourcing this script."
  exit 1
fi

if [ ! -f "$PROJECT_HOME/.env" ]; then
    echo ".env file not found in the project root directory: $PROJECT_HOME."
    exit 1
fi

set -a
source "$PROJECT_HOME/.env"
set +a

if [[ -n "$APP_ENV" && -f "$PROJECT_HOME/.env.$APP_ENV" ]]; then
  echo "Loading environment variables from $PROJECT_HOME/.env.$APP_ENV"
  set -a
  source "$PROJECT_HOME/.env.$APP_ENV"
  set +a
fi