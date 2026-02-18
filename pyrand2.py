#!/usr/bin/env python3
"""Random float generator (0.0-1.0)."""

import random


def main():
    """Generate and print a random float between 0.0 and 1.0."""
    number = random.random()
    print(f"Random float (0.0-1.0): {number:.6f}")


if __name__ == "__main__":
    main()
