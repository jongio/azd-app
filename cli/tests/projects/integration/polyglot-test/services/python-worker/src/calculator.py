"""Calculator module with basic math operations."""


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
    """Divide a by b.
    
    Raises:
        ValueError: If b is zero.
    """
    if b == 0:
        raise ValueError("Division by zero")
    return a / b


def factorial(n: int) -> int:
    """Calculate factorial of n.
    
    Raises:
        ValueError: If n is negative.
    """
    if n < 0:
        raise ValueError("Factorial of negative number")
    if n == 0 or n == 1:
        return 1
    return n * factorial(n - 1)


def fibonacci(n: int) -> int:
    """Calculate the nth Fibonacci number.
    
    Raises:
        ValueError: If n is negative.
    """
    if n < 0:
        raise ValueError("Fibonacci of negative number")
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)


def is_prime(n: int) -> bool:
    """Check if n is a prime number."""
    if n < 2:
        return False
    for i in range(2, int(n ** 0.5) + 1):
        if n % i == 0:
            return False
    return True
