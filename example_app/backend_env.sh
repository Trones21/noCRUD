#!/bin/bash
# An example of the load_backend_env.sh file I keep in ~/env_setup/<project>/
# Check if the script is being sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Please source this script to set the environment variables:"
    echo "  source ${0}"
    exit 1
fi

#Set postgres env vars
 export DB_NAME="example_app_base"
 export DB_USER="postgres"
 export DB_PASS="postgres"
 export DB_HOST="localhost"
 export DB_PORT="5432"
 
 export APP_PORT="8000"
 
# Set S3 env vars
export top_level_bucket=""
export S3_ACCESS_KEY=""
export S3_SECRET_KEY=""


export DJANGO_KEY="django-insecure-er(28vzqa)7_^n+wmqn@)3=1)eup#m*!lr49j!bzdu&!ai$=hu'"
export DJANGO_CORS="http://localhost:9443"

echo "Environment variables have been set."
