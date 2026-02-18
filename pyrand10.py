"""Random number generation utilities with statistical analysis."""

import random
import statistics
from collections import Counter
from typing import List, Tuple


def generate_weighted_random(weights: dict[str, float], count: int) -> List[str]:
    """Generate random choices based on weighted probabilities.

    Args:
        weights: Dictionary mapping choices to their weights
        count: Number of random choices to generate

    Returns:
        List of randomly chosen items according to weights
    """
    choices = list(weights.keys())
    weight_values = list(weights.values())
    return random.choices(choices, weights=weight_values, k=count)


def analyze_distribution(data: List[str]) -> dict[str, float]:
    """Analyze the distribution of random data.

    Args:
        data: List of random choices

    Returns:
        Dictionary mapping each choice to its observed frequency
    """
    total = len(data)
    counts = Counter(data)
    return {item: count / total for item, count in counts.items()}


def monte_carlo_pi(iterations: int = 100_000) -> Tuple[float, float]:
    """Estimate π using Monte Carlo simulation.

    Args:
        iterations: Number of random points to generate

    Returns:
        Tuple of (estimated π, error from actual π)
    """
    inside_circle = sum(
        1 for _ in range(iterations)
        if random.random()**2 + random.random()**2 <= 1.0
    )
    pi_estimate = 4 * inside_circle / iterations
    return pi_estimate, abs(pi_estimate - 3.141592653589793)


if __name__ == "__main__":
    # Demonstrate weighted random generation
    fruit_weights = {"apple": 0.5, "banana": 0.3, "cherry": 0.2}
    samples = generate_weighted_random(fruit_weights, 1000)
    distribution = analyze_distribution(samples)

    print("Weighted Random Sampling:")
    for fruit, freq in distribution.items():
        print(f"  {fruit}: {freq:.3f} (expected: {fruit_weights[fruit]})")

    # Estimate π
    pi_est, error = monte_carlo_pi(100_000)
    print(f"\nMonte Carlo π estimation: {pi_est:.6f} (error: {error:.6f})")
