#Clear Old Migrations & DB
rm -r ./api/migrations/

bash ../python/drop_dbs_by_pattern.sh template_db

#New DB (Uses env var)
python ../python/init_db.py

# Make Migration & Migrate
mkdir ./api/migrations/
touch ./api/migrations/__init__.py

python manage.py makemigrations
echo "--------------Migrations Created------------------------"
python manage.py migrate
echo "--------------Migrations Applied------------------------"