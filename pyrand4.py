#!/usr/bin/env python3
"""Dice roll simulator (1-6)."""

import random


def main():
    """Simulate rolling a six-sided die and print the result."""
    roll = random.randint(1, 6)
    print(f"Dice roll: {roll}")


if __name__ == "__main__":
    main()
