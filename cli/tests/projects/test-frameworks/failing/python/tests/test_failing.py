# Intentionally failing tests to verify error handling
import pytest

class TestFailingTests:
    def test_should_fail_assertion(self):
        """This test should fail with assertion error"""
        assert 1 + 1 == 3, "Math is broken"
    
    def test_should_pass(self):
        """This test should pass"""
        assert 2 + 2 == 4
    
    def test_should_fail_string(self):
        """This test should fail with string mismatch"""
        assert "hello" == "world"
