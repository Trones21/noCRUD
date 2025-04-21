#!/bin/bash
#Please run backend_env to set environment variables, or modify here
# Database connection details
DB_HOST=$POSTGRES_HOST
DB_PORT=$POSTGRES_PORT
DB_NAME=$POSTGRES_DB_NAME
DB_USER=$POSTGRES_USER
DB_PASSWORD=$POSTGRES_PASSWORD


# Default SQL query ( in case it's easier to write here than as flag)
default_query="CREATE TABLE IF NOT EXISTS example_table (id SERIAL PRIMARY KEY, name VARCHAR(100));"

# Parse command-line arguments
while getopts ":q:" opt; do
  case $opt in
    q)
      SQL_QUERY="$OPTARG"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
    :)
      echo "Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

# Use default query if no -q flag is provided
if [ -z "$SQL_QUERY" ]; then
  SQL_QUERY="$default_query"
fi

# Execute the query and capture the output
OUTPUT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "$SQL_QUERY")

# Check if the command was successful
if [ $? -eq 0 ]; then
  echo "Query executed successfully."
  echo "Output:"
  echo "$OUTPUT"
else
  echo -e "Failed to execute query.

    PORT:       $DB_PORT    
    HOST:       $DB_HOST    
    USER:       $DB_USER    
    DB NAME:    $DB_NAME    
    Query:      $DDL_QUERY  
  ----------------------
  Error output:
  $OUTPUT
  "   
fi
