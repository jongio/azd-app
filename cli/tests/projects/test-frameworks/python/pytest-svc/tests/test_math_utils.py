"""
Tests for math utilities using pytest.
"""
import pytest
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from src.math_utils import (
    add, subtract, multiply, divide, power, sqrt, factorial,
    fibonacci, is_prime, gcd, lcm, quadratic_roots, mean, median,
    standard_deviation
)


class TestBasicOperations:
    """Tests for basic arithmetic operations."""

    def test_add(self):
        assert add(2, 3) == 5
        assert add(-1, 1) == 0
        assert add(0, 0) == 0

    def test_subtract(self):
        assert subtract(5, 3) == 2
        assert subtract(3, 5) == -2

    def test_multiply(self):
        assert multiply(3, 4) == 12
        assert multiply(-2, 3) == -6
        assert multiply(0, 100) == 0

    def test_divide(self):
        assert divide(10, 2) == 5
        assert divide(7, 2) == 3.5

    def test_divide_by_zero(self):
        with pytest.raises(ZeroDivisionError):
            divide(10, 0)


class TestPowerAndRoot:
    """Tests for power and root operations."""

    def test_power(self):
        assert power(2, 3) == 8
        assert power(5, 0) == 1
        assert power(2, -1) == 0.5

    def test_sqrt(self):
        assert sqrt(16) == 4
        assert sqrt(0) == 0
        assert abs(sqrt(2) - 1.414) < 0.01

    def test_sqrt_negative(self):
        with pytest.raises(ValueError):
            sqrt(-1)


class TestFactorial:
    """Tests for factorial function."""

    @pytest.mark.parametrize("n,expected", [
        (0, 1),
        (1, 1),
        (5, 120),
        (10, 3628800),
    ])
    def test_factorial(self, n, expected):
        assert factorial(n) == expected

    def test_factorial_negative(self):
        with pytest.raises(ValueError):
            factorial(-1)


class TestFibonacci:
    """Tests for fibonacci function."""

    @pytest.mark.parametrize("n,expected", [
        (0, 0),
        (1, 1),
        (2, 1),
        (10, 55),
        (20, 6765),
    ])
    def test_fibonacci(self, n, expected):
        assert fibonacci(n) == expected

    def test_fibonacci_negative(self):
        with pytest.raises(ValueError):
            fibonacci(-1)


class TestPrimeNumbers:
    """Tests for prime number detection."""

    @pytest.mark.parametrize("n", [2, 3, 5, 7, 11, 13, 97])
    def test_is_prime_true(self, n):
        assert is_prime(n) is True

    @pytest.mark.parametrize("n", [0, 1, 4, 6, 8, 9, 100])
    def test_is_prime_false(self, n):
        assert is_prime(n) is False


class TestGcdLcm:
    """Tests for GCD and LCM functions."""

    def test_gcd(self):
        assert gcd(12, 8) == 4
        assert gcd(17, 13) == 1
        assert gcd(100, 25) == 25

    def test_lcm(self):
        assert lcm(4, 6) == 12
        assert lcm(3, 5) == 15
        assert lcm(12, 18) == 36


class TestQuadratic:
    """Tests for quadratic equation solver."""

    def test_real_roots(self):
        roots = quadratic_roots(1, -5, 6)
        assert set(roots) == {2.0, 3.0}

    def test_complex_roots(self):
        roots = quadratic_roots(1, 0, 1)
        assert roots[0].real == 0
        assert abs(roots[0].imag) == 1

    def test_zero_coefficient(self):
        with pytest.raises(ValueError):
            quadratic_roots(0, 2, 3)


class TestStatistics:
    """Tests for statistical functions."""

    def test_mean(self):
        assert mean([1, 2, 3, 4, 5]) == 3
        assert mean([10]) == 10

    def test_mean_empty(self):
        with pytest.raises(ValueError):
            mean([])

    def test_median_odd(self):
        assert median([1, 2, 3, 4, 5]) == 3

    def test_median_even(self):
        assert median([1, 2, 3, 4]) == 2.5

    def test_standard_deviation(self):
        result = standard_deviation([2, 4, 4, 4, 5, 5, 7, 9])
        assert abs(result - 2.0) < 0.01
