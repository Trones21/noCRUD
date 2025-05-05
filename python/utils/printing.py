def format_crud_print(crud: dict):
    success = "\033[92m✔\033[0m"
    fail = "\033[91m✘\033[0m"
    for k in crud:
        crud[k] = success if crud[k] else fail

    toPrint = (
        f"C:{crud['create']} R:{crud['read']} U:{crud['update']} D:{crud['delete']}"
    )
    return toPrint


def format_req_flow_print(obj):
    return obj


def print_group_separator(text):
    print("\n" + "=" * 80)
    print(f"{text}".center(80))
    print("=" * 80)


def print_warn(text):
    print("\n" + "=" * 33, "WARNING ⚠️ ", "=" * 34)
    print(f"{text}".center(79))
    print("=" * 79)
