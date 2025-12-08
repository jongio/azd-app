/**
 * String utilities module - demonstrates various string operations
 */

export function capitalize(str) {
  if (!str) return '';
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
}

export function reverse(str) {
  return str.split('').reverse().join('');
}

export function truncate(str, maxLength, suffix = '...') {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - suffix.length) + suffix;
}

export function countWords(str) {
  if (!str || !str.trim()) return 0;
  return str.trim().split(/\s+/).length;
}

export function isPalindrome(str) {
  const cleaned = str.toLowerCase().replace(/[^a-z0-9]/g, '');
  return cleaned === cleaned.split('').reverse().join('');
}

export function camelToSnake(str) {
  return str.replace(/([A-Z])/g, '_$1').toLowerCase().replace(/^_/, '');
}

export function snakeToCamel(str) {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

export function slugify(str) {
  return str
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function countOccurrences(str, substring) {
  if (!substring) return 0;
  let count = 0;
  let pos = 0;
  while ((pos = str.indexOf(substring, pos)) !== -1) {
    count++;
    pos += substring.length;
  }
  return count;
}

export function padStart(str, length, char = ' ') {
  if (str.length >= length) return str;
  return char.repeat(length - str.length) + str;
}

export function padEnd(str, length, char = ' ') {
  if (str.length >= length) return str;
  return str + char.repeat(length - str.length);
}

export function removeWhitespace(str) {
  return str.replace(/\s+/g, '');
}

export function isEmail(str) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(str);
}

export function extractNumbers(str) {
  const matches = str.match(/\d+/g);
  return matches ? matches.map(Number) : [];
}

export function wordWrap(str, width) {
  if (str.length <= width) return str;
  const words = str.split(' ');
  const lines = [];
  let currentLine = '';

  for (const word of words) {
    if (currentLine.length + word.length + 1 <= width) {
      currentLine += (currentLine ? ' ' : '') + word;
    } else {
      if (currentLine) lines.push(currentLine);
      currentLine = word;
    }
  }
  if (currentLine) lines.push(currentLine);
  return lines.join('\n');
}
