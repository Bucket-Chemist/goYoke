#!/usr/bin/env python3
"""Basic random integer generator (1-100)."""

import random


def main():
    """Generate and print a random integer between 1 and 100."""
    number = random.randint(1, 100)
    print(f"Random integer (1-100): {number}")


if __name__ == "__main__":
    main()
