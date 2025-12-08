/**
 * Calculator module - demonstrates various math operations
 */

export function add(a, b) {
  return a + b;
}

export function subtract(a, b) {
  return a - b;
}

export function multiply(a, b) {
  return a * b;
}

export function divide(a, b) {
  if (b === 0) {
    throw new Error('Division by zero');
  }
  return a / b;
}

export function power(base, exponent) {
  return Math.pow(base, exponent);
}

export function sqrt(n) {
  if (n < 0) {
    throw new Error('Cannot calculate square root of negative number');
  }
  return Math.sqrt(n);
}

export function modulo(a, b) {
  if (b === 0) {
    throw new Error('Modulo by zero');
  }
  return a % b;
}

export function absolute(n) {
  return Math.abs(n);
}

export function factorial(n) {
  if (n < 0) {
    throw new Error('Cannot calculate factorial of negative number');
  }
  if (n === 0 || n === 1) {
    return 1;
  }
  let result = 1;
  for (let i = 2; i <= n; i++) {
    result *= i;
  }
  return result;
}

export function fibonacci(n) {
  if (n < 0) {
    throw new Error('Cannot calculate fibonacci of negative number');
  }
  if (n <= 1) {
    return n;
  }
  let a = 0, b = 1;
  for (let i = 2; i <= n; i++) {
    [a, b] = [b, a + b];
  }
  return b;
}

export function isPrime(n) {
  if (n < 2) return false;
  if (n === 2) return true;
  if (n % 2 === 0) return false;
  for (let i = 3; i <= Math.sqrt(n); i += 2) {
    if (n % i === 0) return false;
  }
  return true;
}

export function gcd(a, b) {
  a = Math.abs(a);
  b = Math.abs(b);
  while (b !== 0) {
    [a, b] = [b, a % b];
  }
  return a;
}

export function lcm(a, b) {
  return Math.abs(a * b) / gcd(a, b);
}
