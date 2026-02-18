#!/usr/bin/env python3
"""Random password generator - 8 characters."""

import random
import string

if __name__ == "__main__":
    characters = string.ascii_letters + string.digits + string.punctuation
    password = ''.join(random.choice(characters) for _ in range(8))
    print(f"Random password: {password}")
