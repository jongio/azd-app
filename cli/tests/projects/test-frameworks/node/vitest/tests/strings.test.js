import { describe, it, expect } from 'vitest';
import {
  capitalize, reverse, truncate, countWords, isPalindrome,
  camelToSnake, snakeToCamel, slugify, countOccurrences,
  padStart, padEnd, removeWhitespace, isEmail, extractNumbers, wordWrap
} from '../src/strings.js';

describe('String Utilities', () => {
  describe('capitalize', () => {
    it('capitalizes first letter', () => {
      expect(capitalize('hello')).toBe('Hello');
    });

    it('handles empty string', () => {
      expect(capitalize('')).toBe('');
    });

    it('lowercases rest of string', () => {
      expect(capitalize('hELLO')).toBe('Hello');
    });
  });

  describe('reverse', () => {
    it('reverses a string', () => {
      expect(reverse('hello')).toBe('olleh');
    });

    it('handles empty string', () => {
      expect(reverse('')).toBe('');
    });

    it('handles single character', () => {
      expect(reverse('a')).toBe('a');
    });
  });

  describe('truncate', () => {
    it('truncates long strings', () => {
      expect(truncate('hello world', 8)).toBe('hello...');
    });

    it('returns original if shorter than max', () => {
      expect(truncate('hello', 10)).toBe('hello');
    });

    it('uses custom suffix', () => {
      expect(truncate('hello world', 9, '…')).toBe('hello wo…');
    });
  });

  describe('countWords', () => {
    it('counts words in sentence', () => {
      expect(countWords('hello world')).toBe(2);
    });

    it('handles multiple spaces', () => {
      expect(countWords('hello   world')).toBe(2);
    });

    it('returns 0 for empty string', () => {
      expect(countWords('')).toBe(0);
    });
  });

  describe('isPalindrome', () => {
    it('detects palindromes', () => {
      expect(isPalindrome('racecar')).toBe(true);
    });

    it('ignores case and non-alphanumeric', () => {
      expect(isPalindrome('A man, a plan, a canal: Panama')).toBe(true);
    });

    it('detects non-palindromes', () => {
      expect(isPalindrome('hello')).toBe(false);
    });
  });

  describe('camelToSnake', () => {
    it('converts camelCase to snake_case', () => {
      expect(camelToSnake('camelCase')).toBe('camel_case');
    });

    it('handles multiple capitals', () => {
      expect(camelToSnake('myVariableName')).toBe('my_variable_name');
    });
  });

  describe('snakeToCamel', () => {
    it('converts snake_case to camelCase', () => {
      expect(snakeToCamel('snake_case')).toBe('snakeCase');
    });

    it('handles multiple underscores', () => {
      expect(snakeToCamel('my_variable_name')).toBe('myVariableName');
    });
  });

  describe('slugify', () => {
    it('creates URL-safe slug', () => {
      expect(slugify('Hello World!')).toBe('hello-world');
    });

    it('handles special characters', () => {
      expect(slugify('Test@#$String')).toBe('teststring');
    });
  });

  describe('countOccurrences', () => {
    it('counts substring occurrences', () => {
      expect(countOccurrences('banana', 'an')).toBe(2);
    });

    it('returns 0 when not found', () => {
      expect(countOccurrences('hello', 'x')).toBe(0);
    });

    it('handles empty substring', () => {
      expect(countOccurrences('hello', '')).toBe(0);
    });
  });

  describe('padStart/padEnd', () => {
    it('pads start with spaces', () => {
      expect(padStart('5', 3)).toBe('  5');
    });

    it('pads start with custom char', () => {
      expect(padStart('5', 3, '0')).toBe('005');
    });

    it('pads end with spaces', () => {
      expect(padEnd('5', 3)).toBe('5  ');
    });
  });

  describe('removeWhitespace', () => {
    it('removes all whitespace', () => {
      expect(removeWhitespace('hello world')).toBe('helloworld');
    });

    it('removes tabs and newlines', () => {
      expect(removeWhitespace('a\tb\nc')).toBe('abc');
    });
  });

  describe('isEmail', () => {
    it('validates correct emails', () => {
      expect(isEmail('test@example.com')).toBe(true);
    });

    it('rejects invalid emails', () => {
      expect(isEmail('invalid')).toBe(false);
      expect(isEmail('no@domain')).toBe(false);
    });
  });

  describe('extractNumbers', () => {
    it('extracts numbers from string', () => {
      expect(extractNumbers('abc123def456')).toEqual([123, 456]);
    });

    it('returns empty array when no numbers', () => {
      expect(extractNumbers('no numbers')).toEqual([]);
    });
  });

  describe('wordWrap', () => {
    it('wraps text at specified width', () => {
      expect(wordWrap('hello world foo bar', 10)).toBe('hello\nworld foo\nbar');
    });

    it('returns original if shorter than width', () => {
      expect(wordWrap('short', 10)).toBe('short');
    });
  });
});
