import requests

def assert_fail(func, *args, **kwargs):
    """Expect a failure from the given function call."""
    try:
        func(*args, **kwargs)
        raise AssertionError("Expected failure but function succeeded.")
    except requests.exceptions.RequestException as e:
        print(f"Failure as expected: {e}")