// Package calculator provides basic math operations.
package calculator

import (
	"errors"
	"math"
)

// ErrDivisionByZero is returned when dividing by zero.
var ErrDivisionByZero = errors.New("division by zero")

// ErrNegativeInput is returned for operations that don't accept negative numbers.
var ErrNegativeInput = errors.New("negative input not allowed")

// Add returns the sum of two numbers.
func Add(a, b float64) float64 {
	return a + b
}

// Subtract returns a minus b.
func Subtract(a, b float64) float64 {
	return a - b
}

// Multiply returns the product of two numbers.
func Multiply(a, b float64) float64 {
	return a * b
}

// Divide returns a divided by b.
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return a / b, nil
}

// Factorial returns n!
func Factorial(n int) (int, error) {
	if n < 0 {
		return 0, ErrNegativeInput
	}
	if n == 0 || n == 1 {
		return 1, nil
	}
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result, nil
}

// Fibonacci returns the nth Fibonacci number.
func Fibonacci(n int) (int, error) {
	if n < 0 {
		return 0, ErrNegativeInput
	}
	if n <= 1 {
		return n, nil
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b, nil
}

// IsPrime returns true if n is a prime number.
func IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i <= int(math.Sqrt(float64(n))); i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Power returns a raised to the power of b.
func Power(a float64, b int) float64 {
	return math.Pow(a, float64(b))
}

// Abs returns the absolute value of a.
func Abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
