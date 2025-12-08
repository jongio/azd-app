import pytest


def test_greet():
    from src.main import greet
    assert greet("World") == "Hello, World!"


def test_add():
    from src.main import add
    assert add(2, 3) == 5


def test_add_negative():
    from src.main import add
    assert add(-1, 1) == 0
