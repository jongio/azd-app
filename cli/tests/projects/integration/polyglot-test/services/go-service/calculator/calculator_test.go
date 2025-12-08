package calculator

import (
	"testing"
)

func TestUnitAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     float64
		expected float64
	}{
		{"positive numbers", 2, 3, 5},
		{"negative numbers", -2, -3, -5},
		{"with zero", 5, 0, 5},
		{"floats", 1.5, 2.5, 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Add(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestUnitSubtract(t *testing.T) {
	tests := []struct {
		name     string
		a, b     float64
		expected float64
	}{
		{"positive result", 5, 3, 2},
		{"negative result", 3, 5, -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Subtract(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Subtract(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestUnitMultiply(t *testing.T) {
	tests := []struct {
		name     string
		a, b     float64
		expected float64
	}{
		{"positive numbers", 3, 4, 12},
		{"with zero", 5, 0, 0},
		{"negative numbers", -3, 4, -12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Multiply(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Multiply(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestUnitDivide(t *testing.T) {
	result, err := Divide(10, 2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5 {
		t.Errorf("Divide(10, 2) = %v, want 5", result)
	}
}

func TestUnitDivideByZero(t *testing.T) {
	_, err := Divide(10, 0)
	if err != ErrDivisionByZero {
		t.Errorf("Expected ErrDivisionByZero, got %v", err)
	}
}

func TestUnitFactorial(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
		wantErr  bool
	}{
		{"factorial of 5", 5, 120, false},
		{"factorial of 0", 0, 1, false},
		{"factorial of 1", 1, 1, false},
		{"negative number", -1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Factorial(tt.n)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Factorial(%v) = %v, want %v", tt.n, result, tt.expected)
				}
			}
		})
	}
}

func TestUnitFibonacci(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
		wantErr  bool
	}{
		{"fib of 0", 0, 0, false},
		{"fib of 1", 1, 1, false},
		{"fib of 10", 10, 55, false},
		{"negative number", -1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Fibonacci(tt.n)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Fibonacci(%v) = %v, want %v", tt.n, result, tt.expected)
				}
			}
		})
	}
}

func TestUnitIsPrime(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected bool
	}{
		{"2 is prime", 2, true},
		{"7 is prime", 7, true},
		{"4 is not prime", 4, false},
		{"1 is not prime", 1, false},
		{"negative is not prime", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrime(tt.n)
			if result != tt.expected {
				t.Errorf("IsPrime(%v) = %v, want %v", tt.n, result, tt.expected)
			}
		})
	}
}

func TestUnitPower(t *testing.T) {
	result := Power(2, 10)
	if result != 1024 {
		t.Errorf("Power(2, 10) = %v, want 1024", result)
	}
}

func TestUnitAbs(t *testing.T) {
	if Abs(-5) != 5 {
		t.Error("Abs(-5) should be 5")
	}
	if Abs(5) != 5 {
		t.Error("Abs(5) should be 5")
	}
}
