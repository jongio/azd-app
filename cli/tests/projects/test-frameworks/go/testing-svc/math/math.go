// Package math provides mathematical utility functions
package math

import (
	"errors"
	"math"
)

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers
func Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	return a * b
}

// Divide returns the quotient of two floats
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// Power returns base raised to exp
func Power(base, exp float64) float64 {
	return math.Pow(base, exp)
}

// Sqrt returns the square root of x
func Sqrt(x float64) (float64, error) {
	if x < 0 {
		return 0, errors.New("cannot take square root of negative number")
	}
	return math.Sqrt(x), nil
}

// Abs returns the absolute value
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Max returns the larger of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Factorial returns n!
func Factorial(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("factorial of negative number")
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

// GCD returns the greatest common divisor
func GCD(a, b int) int {
	a = Abs(a)
	b = Abs(b)
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// LCM returns the least common multiple
func LCM(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return Abs(a*b) / GCD(a, b)
}

// IsPrime checks if n is prime
func IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := 3; i*i <= n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Fibonacci returns the nth fibonacci number
func Fibonacci(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("fibonacci of negative number")
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

// Sum returns the sum of a slice
func Sum(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Average returns the average of a slice
func Average(nums []float64) (float64, error) {
	if len(nums) == 0 {
		return 0, errors.New("cannot average empty slice")
	}
	var sum float64
	for _, n := range nums {
		sum += n
	}
	return sum / float64(len(nums)), nil
}

// IsEven checks if n is even
func IsEven(n int) bool {
	return n%2 == 0
}

// IsOdd checks if n is odd
func IsOdd(n int) bool {
	return n%2 != 0
}

// Clamp constrains value between min and max
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
