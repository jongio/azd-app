import { describe, it, expect } from 'vitest';
import { add, subtract, multiply, divide, factorial } from '../src/calculator.js';

describe('Calculator unit tests', () => {
  describe('add', () => {
    it('unit: should add two positive numbers', () => {
      expect(add(2, 3)).toBe(5);
    });

    it('unit: should add negative numbers', () => {
      expect(add(-2, -3)).toBe(-5);
    });

    it('unit: should add with zero', () => {
      expect(add(5, 0)).toBe(5);
    });
  });

  describe('subtract', () => {
    it('unit: should subtract two numbers', () => {
      expect(subtract(5, 3)).toBe(2);
    });

    it('unit: should handle negative results', () => {
      expect(subtract(3, 5)).toBe(-2);
    });
  });

  describe('multiply', () => {
    it('unit: should multiply two numbers', () => {
      expect(multiply(3, 4)).toBe(12);
    });

    it('unit: should multiply with zero', () => {
      expect(multiply(5, 0)).toBe(0);
    });

    it('unit: should handle negative numbers', () => {
      expect(multiply(-3, 4)).toBe(-12);
    });
  });

  describe('divide', () => {
    it('unit: should divide two numbers', () => {
      expect(divide(10, 2)).toBe(5);
    });

    it('unit: should handle decimal results', () => {
      expect(divide(10, 4)).toBe(2.5);
    });

    it('unit: should throw on division by zero', () => {
      expect(() => divide(10, 0)).toThrow('Division by zero');
    });
  });

  describe('factorial', () => {
    it('unit: should calculate factorial of 5', () => {
      expect(factorial(5)).toBe(120);
    });

    it('unit: should return 1 for factorial of 0', () => {
      expect(factorial(0)).toBe(1);
    });

    it('unit: should return 1 for factorial of 1', () => {
      expect(factorial(1)).toBe(1);
    });

    it('unit: should throw for negative numbers', () => {
      expect(() => factorial(-1)).toThrow('Factorial of negative number');
    });
  });
});
