import time
import functools
import requests
from typing import Callable, TypeVar

F = TypeVar("F", bound=Callable)


def with_perf(text: str | None = None) -> Callable[[F], F]:
    """Decorator to capture performance of passed function with optional text"""

    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # print(f"[DEBUG] Calling: {func.__name__} (wrapped by with_perf)")
            innerStart = time.perf_counter()
            result = func(*args, **kwargs)
            innerEnd = time.perf_counter()
            duration_ms = (innerEnd - innerStart) * 1000
            label = text or f"Duration of {func.__name__}:"
            print(f"{label} {duration_ms:.2f} ms\n")
            return result

        return wrapper

    if callable(text):
        # print(f"🔍 [DEBUG] @with_perf called directly on function: {text.__name__}")
        return decorator(text)
    else:
        return decorator  # syntax error is just pylance failing


def with_stack_trace(func: F) -> F:
    """Decorator to capture stack trace on request failures."""

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        # print(f"DEBUG] Calling: {func.__name__} (wrapped by with_stack_trace)")
        try:
            return func(*args, **kwargs)
        except requests.exceptions.RequestException:
            print(f"[DEBUG] Exception raised in {func.__name__}")
            raise  # No print/logging here, just re-raise the exception

    return wrapper  # syntax error is just pylance failing


## Original
# def with_stack_trace(func):
#     """Decorator to capture stack trace on request failures."""

#     @functools.wraps(func)
#     def wrapper(*args, **kwargs):
#         try:
#             return func(*args, **kwargs)
#         except requests.exceptions.RequestException:
#             raise  # No print/logging here, just re-raise the exception

#     return wrapper


# def with_perf(text=None):
#     """Decorator to capture performance of passed function"""

#     def decorator(func):
#         @functools.wraps(func)
#         def wrapper(*args, **kwargs):
#             innerStart = time.perf_counter()
#             result = func(*args, **kwargs)
#             innerEnd = time.perf_counter()
#             duration_ms = (innerEnd - innerStart) * 1000
#             label = text or f"Duration of {func.__name__}:"
#             print(f"{label} {duration_ms:.2f} ms\n")
#             return result

#         return wrapper

#     # Handle the case where the decorator is used without arguments
#     if callable(text):
#         return decorator(text)  # Direct Usage @with_perf
#     else:
#         return decorator  # Usage with parenthese @with_perf("some text")
