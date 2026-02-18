#!/usr/bin/env python3
"""Random color hex code generator."""

import random

if __name__ == "__main__":
    color = "#{:06x}".format(random.randint(0, 0xFFFFFF))
    print(f"Random color: {color}")
