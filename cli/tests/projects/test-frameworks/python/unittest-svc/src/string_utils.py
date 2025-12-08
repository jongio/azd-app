"""
String utilities module - demonstrates various string operations.
"""
import re
from typing import List, Optional


def capitalize(text: str) -> str:
    """Capitalize first letter, lowercase rest."""
    if not text:
        return ""
    return text[0].upper() + text[1:].lower()


def reverse(text: str) -> str:
    """Reverse a string."""
    return text[::-1]


def truncate(text: str, max_length: int, suffix: str = "...") -> str:
    """Truncate string to max_length with suffix."""
    if len(text) <= max_length:
        return text
    return text[:max_length - len(suffix)] + suffix


def count_words(text: str) -> int:
    """Count words in a string."""
    if not text or not text.strip():
        return 0
    return len(text.split())


def is_palindrome(text: str) -> bool:
    """Check if string is a palindrome (ignoring case and non-alphanumeric)."""
    cleaned = re.sub(r'[^a-zA-Z0-9]', '', text.lower())
    return cleaned == cleaned[::-1]


def camel_to_snake(text: str) -> str:
    """Convert camelCase to snake_case."""
    result = re.sub(r'([A-Z])', r'_\1', text).lower()
    return result.lstrip('_')


def snake_to_camel(text: str) -> str:
    """Convert snake_case to camelCase."""
    components = text.split('_')
    return components[0] + ''.join(x.title() for x in components[1:])


def slugify(text: str) -> str:
    """Create URL-safe slug from string."""
    text = text.lower().strip()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[\s_-]+', '-', text)
    return text.strip('-')


def count_occurrences(text: str, substring: str) -> int:
    """Count occurrences of substring in text."""
    if not substring:
        return 0
    count = 0
    pos = 0
    while True:
        pos = text.find(substring, pos)
        if pos == -1:
            break
        count += 1
        pos += len(substring)
    return count


def pad_start(text: str, length: int, char: str = ' ') -> str:
    """Pad string at start to reach length."""
    if len(text) >= length:
        return text
    return char * (length - len(text)) + text


def pad_end(text: str, length: int, char: str = ' ') -> str:
    """Pad string at end to reach length."""
    if len(text) >= length:
        return text
    return text + char * (length - len(text))


def remove_whitespace(text: str) -> str:
    """Remove all whitespace from string."""
    return re.sub(r'\s+', '', text)


def is_email(text: str) -> bool:
    """Check if string is a valid email format."""
    pattern = r'^[^\s@]+@[^\s@]+\.[^\s@]+$'
    return bool(re.match(pattern, text))


def extract_numbers(text: str) -> List[int]:
    """Extract all numbers from string."""
    matches = re.findall(r'\d+', text)
    return [int(m) for m in matches]


def word_wrap(text: str, width: int) -> str:
    """Wrap text to specified width."""
    if len(text) <= width:
        return text
    words = text.split()
    lines = []
    current_line = ""
    
    for word in words:
        if len(current_line) + len(word) + 1 <= width:
            current_line += (" " if current_line else "") + word
        else:
            if current_line:
                lines.append(current_line)
            current_line = word
    
    if current_line:
        lines.append(current_line)
    return '\n'.join(lines)
