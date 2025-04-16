#!/bin/bash

# Check if the script is being sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Please source this script to set the environment variables:"
    echo "  source ${0}"
    exit 1
fi

#Set postgres env vars
 export POSTGRES_NAME="postgres"
 export POSTGRES_USER="postgres"
 export POSTGRES_PASSWORD="postgres"
 export POSTGRES_HOST="localhost"
 export POSTGRES_PORT="5432"
 
# Set S3 env vars
export top_level_bucket=""
export S3_ACCESS_KEY=""
export S3_SECRET_KEY=""


export DJANGO_KEY="django-insecure-er(28vzqa)7_^n+wmqn@)3=1)eup#m*!lr49j!bzdu&!ai$=hu'"
export DJANGO_CORS="http://localhost:9443"

echo "Environment variables have been set."