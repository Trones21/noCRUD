import random
import string


# Produce a random string of N chars
def random_string(length):
    return "".join(random.choices(string.ascii_letters + string.digits, k=length))


# If i did want a random string with varying length I would want to ensure
# its never too small if used on a field with a unique constraint
