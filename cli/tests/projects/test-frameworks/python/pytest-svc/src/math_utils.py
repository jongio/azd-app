"""
Math utilities module - demonstrates various mathematical operations.
"""
import math
from typing import List, Tuple


def add(a: float, b: float) -> float:
    """Add two numbers."""
    return a + b


def subtract(a: float, b: float) -> float:
    """Subtract b from a."""
    return a - b


def multiply(a: float, b: float) -> float:
    """Multiply two numbers."""
    return a * b


def divide(a: float, b: float) -> float:
    """Divide a by b. Raises ZeroDivisionError if b is zero."""
    if b == 0:
        raise ZeroDivisionError("Cannot divide by zero")
    return a / b


def power(base: float, exponent: float) -> float:
    """Calculate base raised to exponent."""
    return math.pow(base, exponent)


def sqrt(n: float) -> float:
    """Calculate square root. Raises ValueError for negative numbers."""
    if n < 0:
        raise ValueError("Cannot calculate square root of negative number")
    return math.sqrt(n)


def factorial(n: int) -> int:
    """Calculate factorial. Raises ValueError for negative numbers."""
    if n < 0:
        raise ValueError("Cannot calculate factorial of negative number")
    return math.factorial(n)


def fibonacci(n: int) -> int:
    """Calculate nth Fibonacci number. Raises ValueError for negative numbers."""
    if n < 0:
        raise ValueError("Cannot calculate fibonacci of negative number")
    if n <= 1:
        return n
    a, b = 0, 1
    for _ in range(2, n + 1):
        a, b = b, a + b
    return b


def is_prime(n: int) -> bool:
    """Check if a number is prime."""
    if n < 2:
        return False
    if n == 2:
        return True
    if n % 2 == 0:
        return False
    for i in range(3, int(math.sqrt(n)) + 1, 2):
        if n % i == 0:
            return False
    return True


def gcd(a: int, b: int) -> int:
    """Calculate greatest common divisor."""
    a, b = abs(a), abs(b)
    while b:
        a, b = b, a % b
    return a


def lcm(a: int, b: int) -> int:
    """Calculate least common multiple."""
    return abs(a * b) // gcd(a, b)


def quadratic_roots(a: float, b: float, c: float) -> Tuple[complex, complex]:
    """Calculate roots of quadratic equation ax^2 + bx + c = 0."""
    if a == 0:
        raise ValueError("Coefficient 'a' cannot be zero for quadratic equation")
    discriminant = b * b - 4 * a * c
    if discriminant >= 0:
        sqrt_disc = math.sqrt(discriminant)
        return ((-b + sqrt_disc) / (2 * a), (-b - sqrt_disc) / (2 * a))
    else:
        real = -b / (2 * a)
        imag = math.sqrt(-discriminant) / (2 * a)
        return (complex(real, imag), complex(real, -imag))


def mean(numbers: List[float]) -> float:
    """Calculate arithmetic mean."""
    if not numbers:
        raise ValueError("Cannot calculate mean of empty list")
    return sum(numbers) / len(numbers)


def median(numbers: List[float]) -> float:
    """Calculate median."""
    if not numbers:
        raise ValueError("Cannot calculate median of empty list")
    sorted_nums = sorted(numbers)
    n = len(sorted_nums)
    mid = n // 2
    if n % 2 == 0:
        return (sorted_nums[mid - 1] + sorted_nums[mid]) / 2
    return sorted_nums[mid]


def standard_deviation(numbers: List[float]) -> float:
    """Calculate population standard deviation."""
    if not numbers:
        raise ValueError("Cannot calculate standard deviation of empty list")
    avg = mean(numbers)
    variance = sum((x - avg) ** 2 for x in numbers) / len(numbers)
    return math.sqrt(variance)
