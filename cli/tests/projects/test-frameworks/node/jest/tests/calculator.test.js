import {
  add, subtract, multiply, divide, power, sqrt,
  modulo, absolute, factorial, fibonacci, isPrime, gcd, lcm
} from '../src/calculator.js';

describe('Calculator', () => {
  describe('Basic Operations', () => {
    test('add returns sum of two numbers', () => {
      expect(add(2, 3)).toBe(5);
      expect(add(-1, 1)).toBe(0);
      expect(add(0, 0)).toBe(0);
    });

    test('subtract returns difference of two numbers', () => {
      expect(subtract(5, 3)).toBe(2);
      expect(subtract(3, 5)).toBe(-2);
      expect(subtract(0, 0)).toBe(0);
    });

    test('multiply returns product of two numbers', () => {
      expect(multiply(3, 4)).toBe(12);
      expect(multiply(-2, 3)).toBe(-6);
      expect(multiply(0, 100)).toBe(0);
    });

    test('divide returns quotient of two numbers', () => {
      expect(divide(10, 2)).toBe(5);
      expect(divide(7, 2)).toBe(3.5);
      expect(divide(-6, 2)).toBe(-3);
    });

    test('divide throws error for division by zero', () => {
      expect(() => divide(10, 0)).toThrow('Division by zero');
    });
  });

  describe('Power and Root', () => {
    test('power calculates exponentiation', () => {
      expect(power(2, 3)).toBe(8);
      expect(power(5, 0)).toBe(1);
      expect(power(2, -1)).toBe(0.5);
    });

    test('sqrt calculates square root', () => {
      expect(sqrt(16)).toBe(4);
      expect(sqrt(0)).toBe(0);
      expect(sqrt(2)).toBeCloseTo(1.414, 2);
    });

    test('sqrt throws error for negative numbers', () => {
      expect(() => sqrt(-1)).toThrow('Cannot calculate square root of negative number');
    });
  });

  describe('Modulo and Absolute', () => {
    test('modulo returns remainder', () => {
      expect(modulo(10, 3)).toBe(1);
      expect(modulo(15, 5)).toBe(0);
    });

    test('modulo throws error for modulo by zero', () => {
      expect(() => modulo(10, 0)).toThrow('Modulo by zero');
    });

    test('absolute returns absolute value', () => {
      expect(absolute(-5)).toBe(5);
      expect(absolute(5)).toBe(5);
      expect(absolute(0)).toBe(0);
    });
  });

  describe('Factorial', () => {
    test('factorial calculates correctly', () => {
      expect(factorial(0)).toBe(1);
      expect(factorial(1)).toBe(1);
      expect(factorial(5)).toBe(120);
      expect(factorial(10)).toBe(3628800);
    });

    test('factorial throws error for negative numbers', () => {
      expect(() => factorial(-1)).toThrow('Cannot calculate factorial of negative number');
    });
  });

  describe('Fibonacci', () => {
    test('fibonacci returns correct sequence values', () => {
      expect(fibonacci(0)).toBe(0);
      expect(fibonacci(1)).toBe(1);
      expect(fibonacci(2)).toBe(1);
      expect(fibonacci(10)).toBe(55);
    });

    test('fibonacci throws error for negative numbers', () => {
      expect(() => fibonacci(-1)).toThrow('Cannot calculate fibonacci of negative number');
    });
  });

  describe('Prime Numbers', () => {
    test('isPrime correctly identifies primes', () => {
      expect(isPrime(2)).toBe(true);
      expect(isPrime(7)).toBe(true);
      expect(isPrime(13)).toBe(true);
      expect(isPrime(97)).toBe(true);
    });

    test('isPrime correctly identifies non-primes', () => {
      expect(isPrime(0)).toBe(false);
      expect(isPrime(1)).toBe(false);
      expect(isPrime(4)).toBe(false);
      expect(isPrime(100)).toBe(false);
    });
  });

  describe('GCD and LCM', () => {
    test('gcd calculates greatest common divisor', () => {
      expect(gcd(12, 8)).toBe(4);
      expect(gcd(17, 13)).toBe(1);
      expect(gcd(100, 25)).toBe(25);
    });

    test('lcm calculates least common multiple', () => {
      expect(lcm(4, 6)).toBe(12);
      expect(lcm(3, 5)).toBe(15);
      expect(lcm(12, 18)).toBe(36);
    });
  });
});
