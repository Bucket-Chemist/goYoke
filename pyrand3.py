#!/usr/bin/env python3
"""Random choice from a list."""

import random


def main():
    """Choose and print a random item from a predefined list."""
    choices = ["apple", "banana", "cherry", "date", "elderberry"]
    selected = random.choice(choices)
    print(f"Random choice from {choices}: {selected}")


if __name__ == "__main__":
    main()
