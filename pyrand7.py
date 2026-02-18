#!/usr/bin/env python3
"""Random date generator - within last year."""

import random
from datetime import datetime, timedelta

if __name__ == "__main__":
    end_date = datetime.now()
    start_date = end_date - timedelta(days=365)
    random_days = random.randint(0, 365)
    random_date = start_date + timedelta(days=random_days)
    print(f"Random date: {random_date.strftime('%Y-%m-%d')}")
