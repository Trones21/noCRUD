import subprocess
import psycopg
from psycopg import sql
from typing import Optional
from config import APP_DIR
from utils.fixtures import addFixtures
import sys
import os

# global => file local
app_dir = APP_DIR


class DBClient:
    def __init__(self):
        self.dbname = os.getenv("POSTGRES_DB_NAME", "postgres")
        self.user = os.getenv("POSTGRES_USER", "postgres")
        self.pw = os.getenv("POSTGRES_PASSWORD", "postgres")
        self.host = os.getenv("POSTGRES_HOST", "localhost")
        self.port = os.getenv("POSTGRES_PORT", "5432")

        # Connect to the default DB to run CREATE DATABASE
        self.conn = psycopg.connect(
            dbname=self.dbname,
            user=self.user,
            password=self.pw,
            host=self.host,
            port=self.port,
        )
        self.conn.autocommit = True  # Required for CREATE DATABASE
        self.cur = self.conn.cursor()

    def createDB(self, db_name):
        try:
            self.cur.execute(
                sql.SQL("CREATE DATABASE {}").format(sql.Identifier(db_name))
            )
            print(f"✅ Database '{db_name}' created.")
        except psycopg.errors.DuplicateDatabase:
            print(f"⚠️ Database '{db_name}' already exists.")
        except Exception as e:
            print(f"❌ Failed to create database '{db_name}': {e}")
            raise

    def dropDB(self, db_name):
        try:
            self.cur.execute(
                sql.SQL("DROP DATABASE IF EXISTS {}").format(sql.Identifier(db_name))
            )
            print(f"🗑️  Database '{db_name}' dropped.")
        except Exception as e:
            print(f"❌ Failed to drop database '{db_name}': {e}")
            raise

    def clearTable(self, table):
        with self.conn.cursor() as cur:
            query = sql.SQL("DELETE FROM {};").format(sql.Identifier(table))
            cur.execute(query)
            self.conn.commit()

    def clearAllTables(self):
        """Clears data from all tables in DB but does not drop. (or clear the django default tables)"""
        with self.conn.cursor() as cursor:
            # Get all table names
            cursor.execute("""
                SELECT tablename 
                FROM pg_tables 
                WHERE schemaname = 'public'
                    AND tablename NOT LIKE 'django_%'
                    AND tablename NOT LIKE 'auth_%'
            """)
            tables = [row[0] for row in cursor.fetchall()]

            for _ in range(3):  # Try multiple passes
                for table in tables:
                    try:
                        query = sql.SQL("DELETE FROM {} CASCADE;").format(
                            sql.Identifier(table)
                        )
                        cursor.execute(query)
                        self.conn.commit()
                    except psycopg.Error:
                        pass  # Some tables may still have dependencies, try again later

            cursor.close()

    def verifyAllTablesCleared(self):
        """Verifies clearAllTables ran successfully"""
        cursor = self.conn.cursor()

        cursor.execute("""
            SELECT tablename 
            FROM pg_tables 
            WHERE schemaname = 'public'
                AND tablename NOT LIKE 'django_%'
                AND tablename NOT LIKE 'auth_%'
        """)
        tables = [row[0] for row in cursor.fetchall()]

        for table in tables:
            query = sql.SQL("SELECT COUNT(*) FROM {};").format(sql.Identifier(table))
            cursor.execute(query)
            row = cursor.fetchone()
            count = row[0] if row else 0
            if count > 0:
                print(f"⚠️ Table {table} still has {count} records!")
                sys.exit(1)

        cursor.close()

    def dropAllTables(self):
        """Drops all tables in the 'public' schema"""
        query_str = """
            DO $$ 
            DECLARE 
                r RECORD;
            BEGIN 
                FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
                    EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
                END LOOP;
            END $$;
        """

        with self.conn.cursor() as cur:
            cur.execute(query_str)
            self.conn.commit()

        print("tables dropped")

    def close(self):
        self.conn.close()

    def reset(self, fixturesToAdd: Optional[list[str]] = None) -> None:
        """Drops all tables, re-runs migrate and loads specified set of fixtures (or minmimal if param is none). The order of fixtures matters."""
        self.dropAllTables()

        subprocess.run(
            ["python", "manage.py", "migrate"], cwd=app_dir, check=True
        )  # Python subprocesses auto block so no need to use process.wait()
        addFixtures()
        return

    def clearDataExceptMinimalFixtures(self):
        """Wrapper for db.clearAllTables and re add fixtures. Faster than resetDB because we are not running migrations"""
        self.clearAllTables()
        self.verifyAllTablesCleared()
        print("All tables cleared (except django required)")
        addFixtures()
        return
