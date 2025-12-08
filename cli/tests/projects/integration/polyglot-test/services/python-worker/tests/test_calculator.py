"""Unit tests for calculator module."""

import pytest
from src.calculator import add, subtract, multiply, divide, factorial, fibonacci, is_prime


class TestAdd:
    """Tests for add function."""
    
    @pytest.mark.unit
    def test_add_positive_numbers(self):
        assert add(2, 3) == 5
    
    @pytest.mark.unit
    def test_add_negative_numbers(self):
        assert add(-2, -3) == -5
    
    @pytest.mark.unit
    def test_add_with_zero(self):
        assert add(5, 0) == 5
    
    @pytest.mark.unit
    def test_add_floats(self):
        assert add(1.5, 2.5) == 4.0


class TestSubtract:
    """Tests for subtract function."""
    
    @pytest.mark.unit
    def test_subtract_positive_numbers(self):
        assert subtract(5, 3) == 2
    
    @pytest.mark.unit
    def test_subtract_negative_result(self):
        assert subtract(3, 5) == -2


class TestMultiply:
    """Tests for multiply function."""
    
    @pytest.mark.unit
    def test_multiply_positive_numbers(self):
        assert multiply(3, 4) == 12
    
    @pytest.mark.unit
    def test_multiply_with_zero(self):
        assert multiply(5, 0) == 0
    
    @pytest.mark.unit
    def test_multiply_negative_numbers(self):
        assert multiply(-3, 4) == -12


class TestDivide:
    """Tests for divide function."""
    
    @pytest.mark.unit
    def test_divide_even(self):
        assert divide(10, 2) == 5
    
    @pytest.mark.unit
    def test_divide_decimal_result(self):
        assert divide(10, 4) == 2.5
    
    @pytest.mark.unit
    def test_divide_by_zero_raises(self):
        with pytest.raises(ValueError, match="Division by zero"):
            divide(10, 0)


class TestFactorial:
    """Tests for factorial function."""
    
    @pytest.mark.unit
    def test_factorial_of_5(self):
        assert factorial(5) == 120
    
    @pytest.mark.unit
    def test_factorial_of_0(self):
        assert factorial(0) == 1
    
    @pytest.mark.unit
    def test_factorial_of_1(self):
        assert factorial(1) == 1
    
    @pytest.mark.unit
    def test_factorial_negative_raises(self):
        with pytest.raises(ValueError, match="Factorial of negative"):
            factorial(-1)


class TestFibonacci:
    """Tests for fibonacci function."""
    
    @pytest.mark.unit
    def test_fibonacci_of_0(self):
        assert fibonacci(0) == 0
    
    @pytest.mark.unit
    def test_fibonacci_of_1(self):
        assert fibonacci(1) == 1
    
    @pytest.mark.unit
    def test_fibonacci_of_10(self):
        assert fibonacci(10) == 55
    
    @pytest.mark.unit
    def test_fibonacci_negative_raises(self):
        with pytest.raises(ValueError, match="Fibonacci of negative"):
            fibonacci(-1)


class TestIsPrime:
    """Tests for is_prime function."""
    
    @pytest.mark.unit
    def test_is_prime_2(self):
        assert is_prime(2) is True
    
    @pytest.mark.unit
    def test_is_prime_7(self):
        assert is_prime(7) is True
    
    @pytest.mark.unit
    def test_is_prime_4(self):
        assert is_prime(4) is False
    
    @pytest.mark.unit
    def test_is_prime_1(self):
        assert is_prime(1) is False
    
    @pytest.mark.unit
    def test_is_prime_negative(self):
        assert is_prime(-5) is False
