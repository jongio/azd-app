"""
Tests for string utilities using unittest.
"""
import unittest
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from src.string_utils import (
    capitalize, reverse, truncate, count_words, is_palindrome,
    camel_to_snake, snake_to_camel, slugify, count_occurrences,
    pad_start, pad_end, remove_whitespace, is_email, extract_numbers,
    word_wrap
)


class TestCapitalize(unittest.TestCase):
    """Tests for capitalize function."""

    def test_capitalize_word(self):
        self.assertEqual(capitalize("hello"), "Hello")

    def test_capitalize_empty(self):
        self.assertEqual(capitalize(""), "")

    def test_capitalize_all_caps(self):
        self.assertEqual(capitalize("HELLO"), "Hello")


class TestReverse(unittest.TestCase):
    """Tests for reverse function."""

    def test_reverse_word(self):
        self.assertEqual(reverse("hello"), "olleh")

    def test_reverse_empty(self):
        self.assertEqual(reverse(""), "")

    def test_reverse_single_char(self):
        self.assertEqual(reverse("a"), "a")


class TestTruncate(unittest.TestCase):
    """Tests for truncate function."""

    def test_truncate_long_string(self):
        self.assertEqual(truncate("hello world", 8), "hello...")

    def test_truncate_short_string(self):
        self.assertEqual(truncate("hello", 10), "hello")

    def test_truncate_custom_suffix(self):
        self.assertEqual(truncate("hello world", 9, "…"), "hello wo…")


class TestCountWords(unittest.TestCase):
    """Tests for count_words function."""

    def test_count_words_sentence(self):
        self.assertEqual(count_words("hello world"), 2)

    def test_count_words_empty(self):
        self.assertEqual(count_words(""), 0)

    def test_count_words_whitespace(self):
        self.assertEqual(count_words("   "), 0)


class TestIsPalindrome(unittest.TestCase):
    """Tests for is_palindrome function."""

    def test_palindrome_simple(self):
        self.assertTrue(is_palindrome("racecar"))

    def test_palindrome_sentence(self):
        self.assertTrue(is_palindrome("A man, a plan, a canal: Panama"))

    def test_not_palindrome(self):
        self.assertFalse(is_palindrome("hello"))


class TestCamelSnake(unittest.TestCase):
    """Tests for case conversion functions."""

    def test_camel_to_snake(self):
        self.assertEqual(camel_to_snake("camelCase"), "camel_case")
        self.assertEqual(camel_to_snake("myVariableName"), "my_variable_name")

    def test_snake_to_camel(self):
        self.assertEqual(snake_to_camel("snake_case"), "snakeCase")
        self.assertEqual(snake_to_camel("my_variable_name"), "myVariableName")


class TestSlugify(unittest.TestCase):
    """Tests for slugify function."""

    def test_slugify_basic(self):
        self.assertEqual(slugify("Hello World!"), "hello-world")

    def test_slugify_special_chars(self):
        self.assertEqual(slugify("Test@#$String"), "teststring")


class TestCountOccurrences(unittest.TestCase):
    """Tests for count_occurrences function."""

    def test_count_multiple(self):
        self.assertEqual(count_occurrences("banana", "an"), 2)

    def test_count_none(self):
        self.assertEqual(count_occurrences("hello", "x"), 0)

    def test_count_empty_substring(self):
        self.assertEqual(count_occurrences("hello", ""), 0)


class TestPadding(unittest.TestCase):
    """Tests for padding functions."""

    def test_pad_start_spaces(self):
        self.assertEqual(pad_start("5", 3), "  5")

    def test_pad_start_custom(self):
        self.assertEqual(pad_start("5", 3, "0"), "005")

    def test_pad_end(self):
        self.assertEqual(pad_end("5", 3), "5  ")


class TestRemoveWhitespace(unittest.TestCase):
    """Tests for remove_whitespace function."""

    def test_remove_spaces(self):
        self.assertEqual(remove_whitespace("hello world"), "helloworld")

    def test_remove_tabs_newlines(self):
        self.assertEqual(remove_whitespace("a\tb\nc"), "abc")


class TestIsEmail(unittest.TestCase):
    """Tests for is_email function."""

    def test_valid_email(self):
        self.assertTrue(is_email("test@example.com"))

    def test_invalid_email(self):
        self.assertFalse(is_email("invalid"))
        self.assertFalse(is_email("no@domain"))


class TestExtractNumbers(unittest.TestCase):
    """Tests for extract_numbers function."""

    def test_extract_numbers(self):
        self.assertEqual(extract_numbers("abc123def456"), [123, 456])

    def test_no_numbers(self):
        self.assertEqual(extract_numbers("no numbers"), [])


class TestWordWrap(unittest.TestCase):
    """Tests for word_wrap function."""

    def test_wrap_text(self):
        result = word_wrap("hello world foo bar", 10)
        self.assertEqual(result, "hello\nworld foo\nbar")

    def test_short_text(self):
        self.assertEqual(word_wrap("short", 10), "short")


if __name__ == '__main__':
    unittest.main()
