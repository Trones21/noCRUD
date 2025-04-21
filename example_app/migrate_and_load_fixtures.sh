#!/bin/sh

# Minimum set of fixtures required to make things work
declare -a fixtures=("users.json")

# Create log directory
mkdir -p logs

#Clear Old Migrations & DB
rm -r ./api/migrations/
sh ./clear_db.sh


# Make Migration & Migrate
mkdir ./api/migrations/
touch ./api/migrations/__init__.py

python manage.py makemigrations
echo "--------------Migrations Created------------------------"
python manage.py migrate
echo "--------------Migrations Applied------------------------"

fixture_path= "./api/fixtures/"

for n in $(seq 1 2); do
    echo "--------------Fixtures------------------------"
    for fixture in ${fixtures[@]}
    do
        python manage.py loaddata "${fixture_path}$fixture"
    done
done

read -p "Press Enter to finish" < /dev/tty