#!/usr/bin/env python3
import argparse
import os

TEMPLATE = """from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup


def crud():
    api = setup()
    update_details: UpdateDetails = {{
        "fieldName": "{update_field}",
        "length": 49,
        "newValue": "{field_value}",
    }}
    crud_res = crud_exec("{obj_name}", api, create, update_details)
    return crud_res


def create(api: APIClient):
    \"\"\"
    Create the {obj_name}.
    \"\"\"
    {obj_name} = get_fixture_by_index("{obj_name}s.json", 0)
    res = api.create_object("{obj_name}", {obj_name})
    {obj_name}_id = res["id"]
    return {obj_name}_id
"""


def main():
    parser = argparse.ArgumentParser(
        description="Generate a CRUD flow file with an update step."
    )
    parser.add_argument(
        "obj_name",
        help="The base name of the object (also used for the URL and fixture filename)",
    )
    parser.add_argument(
        "-uf", "--updateField", required=True, help="The field to update in the object"
    )
    parser.add_argument(
        "-ufv", "--fieldval", required=True, help="The new value to set for the field"
    )
    parser.add_argument(
        "-path", default="crud", help="Relative subpath from /flows/ to write the file"
    )

    args = parser.parse_args()

    # File creation
    base_dir = os.path.join("flows", args.path)
    os.makedirs(base_dir, exist_ok=True)

    filename = f"{args.obj_name}.py"
    filepath = os.path.join(base_dir, filename)

    contents = TEMPLATE.format(
        obj_name=args.obj_name, update_field=args.updateField, field_value=args.fieldval
    )

    with open(filepath, "w") as f:
        f.write(contents)

    print(f"✅ File created: {filepath}")


if __name__ == "__main__":
    main()
