package math

import (
	"testing"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 2, 3, 5},
		{"negative numbers", -2, -3, -5},
		{"mixed numbers", -2, 3, 1},
		{"zeros", 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Add(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	if got := Subtract(5, 3); got != 2 {
		t.Errorf("Subtract(5, 3) = %d; want 2", got)
	}
	if got := Subtract(3, 5); got != -2 {
		t.Errorf("Subtract(3, 5) = %d; want -2", got)
	}
}

func TestMultiply(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{2, 3, 6},
		{-2, 3, -6},
		{0, 5, 0},
	}

	for _, tc := range tests {
		if got := Multiply(tc.a, tc.b); got != tc.expected {
			t.Errorf("Multiply(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.expected)
		}
	}
}

func TestDivide(t *testing.T) {
	result, err := Divide(10, 2)
	if err != nil || result != 5 {
		t.Errorf("Divide(10, 2) = %f, %v; want 5, nil", result, err)
	}

	_, err = Divide(10, 0)
	if err == nil {
		t.Error("Divide by zero should return error")
	}
}

func TestPower(t *testing.T) {
	if got := Power(2, 3); got != 8 {
		t.Errorf("Power(2, 3) = %f; want 8", got)
	}
	if got := Power(2, 0); got != 1 {
		t.Errorf("Power(2, 0) = %f; want 1", got)
	}
}

func TestSqrt(t *testing.T) {
	result, err := Sqrt(16)
	if err != nil || result != 4 {
		t.Errorf("Sqrt(16) = %f, %v; want 4, nil", result, err)
	}

	_, err = Sqrt(-1)
	if err == nil {
		t.Error("Sqrt of negative should return error")
	}
}

func TestAbs(t *testing.T) {
	if got := Abs(-5); got != 5 {
		t.Errorf("Abs(-5) = %d; want 5", got)
	}
	if got := Abs(5); got != 5 {
		t.Errorf("Abs(5) = %d; want 5", got)
	}
	if got := Abs(0); got != 0 {
		t.Errorf("Abs(0) = %d; want 0", got)
	}
}

func TestMaxMin(t *testing.T) {
	if got := Max(5, 3); got != 5 {
		t.Errorf("Max(5, 3) = %d; want 5", got)
	}
	if got := Min(5, 3); got != 3 {
		t.Errorf("Min(5, 3) = %d; want 3", got)
	}
}

func TestFactorial(t *testing.T) {
	tests := []struct {
		n        int
		expected int
		hasError bool
	}{
		{0, 1, false},
		{1, 1, false},
		{5, 120, false},
		{-1, 0, true},
	}

	for _, tc := range tests {
		result, err := Factorial(tc.n)
		if tc.hasError && err == nil {
			t.Errorf("Factorial(%d) should return error", tc.n)
		}
		if !tc.hasError && result != tc.expected {
			t.Errorf("Factorial(%d) = %d; want %d", tc.n, result, tc.expected)
		}
	}
}

func TestGCD(t *testing.T) {
	if got := GCD(12, 8); got != 4 {
		t.Errorf("GCD(12, 8) = %d; want 4", got)
	}
	if got := GCD(17, 13); got != 1 {
		t.Errorf("GCD(17, 13) = %d; want 1", got)
	}
}

func TestLCM(t *testing.T) {
	if got := LCM(4, 6); got != 12 {
		t.Errorf("LCM(4, 6) = %d; want 12", got)
	}
	if got := LCM(0, 5); got != 0 {
		t.Errorf("LCM(0, 5) = %d; want 0", got)
	}
}

func TestIsPrime(t *testing.T) {
	primes := []int{2, 3, 5, 7, 11, 13, 17, 19}
	nonPrimes := []int{0, 1, 4, 6, 8, 9, 10}

	for _, p := range primes {
		if !IsPrime(p) {
			t.Errorf("IsPrime(%d) = false; want true", p)
		}
	}

	for _, np := range nonPrimes {
		if IsPrime(np) {
			t.Errorf("IsPrime(%d) = true; want false", np)
		}
	}
}

func TestFibonacci(t *testing.T) {
	expected := []int{0, 1, 1, 2, 3, 5, 8, 13}
	for i, exp := range expected {
		result, err := Fibonacci(i)
		if err != nil || result != exp {
			t.Errorf("Fibonacci(%d) = %d, %v; want %d, nil", i, result, err, exp)
		}
	}

	_, err := Fibonacci(-1)
	if err == nil {
		t.Error("Fibonacci of negative should return error")
	}
}

func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3, 4, 5}); got != 15 {
		t.Errorf("Sum([1,2,3,4,5]) = %d; want 15", got)
	}
	if got := Sum([]int{}); got != 0 {
		t.Errorf("Sum([]) = %d; want 0", got)
	}
}

func TestAverage(t *testing.T) {
	result, err := Average([]float64{1, 2, 3, 4, 5})
	if err != nil || result != 3 {
		t.Errorf("Average([1,2,3,4,5]) = %f, %v; want 3, nil", result, err)
	}

	_, err = Average([]float64{})
	if err == nil {
		t.Error("Average of empty slice should return error")
	}
}

func TestIsEvenIsOdd(t *testing.T) {
	if !IsEven(4) {
		t.Error("IsEven(4) should be true")
	}
	if IsEven(3) {
		t.Error("IsEven(3) should be false")
	}
	if !IsOdd(3) {
		t.Error("IsOdd(3) should be true")
	}
	if IsOdd(4) {
		t.Error("IsOdd(4) should be false")
	}
}

func TestClamp(t *testing.T) {
	if got := Clamp(5, 0, 10); got != 5 {
		t.Errorf("Clamp(5, 0, 10) = %d; want 5", got)
	}
	if got := Clamp(-5, 0, 10); got != 0 {
		t.Errorf("Clamp(-5, 0, 10) = %d; want 0", got)
	}
	if got := Clamp(15, 0, 10); got != 10 {
		t.Errorf("Clamp(15, 0, 10) = %d; want 10", got)
	}
}
