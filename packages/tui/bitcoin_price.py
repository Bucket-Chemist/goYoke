#!/usr/bin/env python3
"""Fetch and display the current Bitcoin price in USD."""

import sys
from typing import Final

import requests

API_URL: Final = "https://api.coingecko.com/api/v3/simple/price"
TIMEOUT: Final = 10.0


def fetch_bitcoin_price() -> float:
    """Fetch the current Bitcoin price in USD from CoinGecko API.

    Returns:
        Current Bitcoin price in USD

    Raises:
        requests.RequestException: If the API request fails
        ValueError: If the API response is invalid
    """
    params = {"ids": "bitcoin", "vs_currencies": "usd"}

    response = requests.get(API_URL, params=params, timeout=TIMEOUT)
    response.raise_for_status()

    data = response.json()

    if "bitcoin" not in data or "usd" not in data["bitcoin"]:
        raise ValueError("Invalid API response format")

    return float(data["bitcoin"]["usd"])


def main() -> int:
    """Main entry point for the script.

    Returns:
        Exit code (0 for success, 1 for failure)
    """
    try:
        price = fetch_bitcoin_price()
        print(f"Current Bitcoin price: ${price:,.2f} USD")
        return 0

    except requests.Timeout:
        print("Error: Request timed out. Please check your internet connection.", file=sys.stderr)
        return 1

    except requests.RequestException as e:
        print(f"Error: Failed to fetch Bitcoin price: {e}", file=sys.stderr)
        return 1

    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1

    except KeyboardInterrupt:
        print("\nOperation cancelled by user.", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
