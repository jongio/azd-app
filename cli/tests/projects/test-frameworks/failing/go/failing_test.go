package failing

import "testing"

func TestShouldFailAssertion(t *testing.T) {
	// This test should fail
	result := 1 + 1
	expected := 3
	if result != expected {
		t.Errorf("Expected %d but got %d", expected, result)
	}
}

func TestShouldPass(t *testing.T) {
	// This test should pass
	result := 2 + 2
	expected := 4
	if result != expected {
		t.Errorf("Expected %d but got %d", expected, result)
	}
}

func TestShouldFailString(t *testing.T) {
	// This test should fail
	result := "hello"
	expected := "world"
	if result != expected {
		t.Errorf("Expected %q but got %q", expected, result)
	}
}
