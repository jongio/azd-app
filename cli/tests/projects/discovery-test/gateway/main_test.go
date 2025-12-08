package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
}

func TestAddNegative(t *testing.T) {
	result := Add(-1, 1)
	if result != 0 {
		t.Errorf("Add(-1, 1) = %d; want 0", result)
	}
}

func TestGreet(t *testing.T) {
	result := Greet("World")
	expected := "Hello, World!"
	if result != expected {
		t.Errorf("Greet(World) = %s; want %s", result, expected)
	}
}
