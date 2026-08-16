from pathlib import Path

# DO NOT ADD A TRAILING SLASH TO EITHER OF THESE

# This will change depending on where your runner lives relative to your app.
# The default points at the bundled example app; in your own project this is
# wherever your backend lives.
APP_DIR = Path(__file__).resolve().parents[1] / "examples" / "example_app"

# Use the full path, not app_dir relative (unless you put that var in here like so)
FIXTURES_PATH = f"{APP_DIR}/api/fixtures"
