#!/usr/bin/env python3
"""Fetch and display the current Ethereum (ETH) price in USD."""

import sys
from urllib.request import urlopen
from urllib.error import URLError
import json


def fetch_eth_price() -> float:
    """Fetch the current Ethereum price from CoinGecko API.

    Returns:
        Current ETH price in USD

    Raises:
        URLError: If the API request fails
        ValueError: If the response is invalid
    """
    url = "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"

    try:
        with urlopen(url, timeout=10) as response:
            data = json.loads(response.read().decode())
            return float(data["ethereum"]["usd"])
    except URLError as e:
        raise URLError(f"Failed to connect to CoinGecko API: {e}") from e
    except (KeyError, ValueError) as e:
        raise ValueError(f"Invalid API response format: {e}") from e


def main() -> int:
    """Main entry point."""
    try:
        price = fetch_eth_price()
        print(f"Current Ethereum (ETH) price: ${price:,.2f} USD")
        return 0
    except URLError as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
