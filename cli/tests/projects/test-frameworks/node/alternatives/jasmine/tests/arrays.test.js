import {
  unique, flatten, chunk, sum, average, min, max,
  compact, intersection, difference, union,
  groupBy, countBy, pluck, sortBy, partition, rotate
} from '../../src/arrays.js';

describe('Array Utilities - Jasmine Tests', () => {
  describe('unique()', () => {
    it('should remove duplicate values', () => {
      expect(unique([1, 2, 2, 3, 3, 3])).toEqual([1, 2, 3]);
    });

    it('should handle empty arrays', () => {
      expect(unique([])).toEqual([]);
    });

    it('should preserve order', () => {
      expect(unique([3, 1, 2, 1, 3])).toEqual([3, 1, 2]);
    });
  });

  describe('flatten()', () => {
    it('should flatten nested arrays', () => {
      expect(flatten([1, [2, [3, [4]]]])).toEqual([1, 2, 3, 4]);
    });

    it('should handle already flat arrays', () => {
      expect(flatten([1, 2, 3])).toEqual([1, 2, 3]);
    });
  });

  describe('chunk()', () => {
    it('should split array into chunks', () => {
      expect(chunk([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]]);
    });

    it('should handle size larger than array', () => {
      expect(chunk([1, 2], 5)).toEqual([[1, 2]]);
    });

    it('should return empty array for invalid size', () => {
      expect(chunk([1, 2, 3], 0)).toEqual([]);
    });
  });

  describe('sum()', () => {
    it('should sum all numbers', () => {
      expect(sum([1, 2, 3, 4])).toEqual(10);
    });

    it('should return 0 for empty array', () => {
      expect(sum([])).toEqual(0);
    });
  });

  describe('average()', () => {
    it('should calculate average', () => {
      expect(average([1, 2, 3, 4, 5])).toEqual(3);
    });

    it('should return 0 for empty array', () => {
      expect(average([])).toEqual(0);
    });
  });

  describe('min() and max()', () => {
    it('should find minimum value', () => {
      expect(min([3, 1, 4, 1, 5])).toEqual(1);
    });

    it('should find maximum value', () => {
      expect(max([3, 1, 4, 1, 5])).toEqual(5);
    });

    it('should return undefined for empty array', () => {
      expect(min([])).toBeUndefined();
      expect(max([])).toBeUndefined();
    });
  });

  describe('compact()', () => {
    it('should remove falsy values', () => {
      expect(compact([0, 1, false, 2, '', 3, null, undefined])).toEqual([1, 2, 3]);
    });
  });

  describe('set operations', () => {
    it('should find intersection', () => {
      expect(intersection([1, 2, 3], [2, 3, 4])).toEqual([2, 3]);
    });

    it('should find difference', () => {
      expect(difference([1, 2, 3], [2, 3, 4])).toEqual([1]);
    });

    it('should find union', () => {
      expect(union([1, 2], [2, 3])).toEqual([1, 2, 3]);
    });
  });

  describe('groupBy()', () => {
    it('should group by key', () => {
      const data = [
        { type: 'a', value: 1 },
        { type: 'b', value: 2 },
        { type: 'a', value: 3 }
      ];
      const result = groupBy(data, 'type');
      expect(result.a.length).toEqual(2);
      expect(result.b.length).toEqual(1);
    });
  });

  describe('countBy()', () => {
    it('should count by key', () => {
      const data = [{ status: 'active' }, { status: 'inactive' }, { status: 'active' }];
      expect(countBy(data, 'status')).toEqual({ active: 2, inactive: 1 });
    });

    it('should count by function', () => {
      expect(countBy([1, 2, 3, 4, 5], n => n % 2 === 0 ? 'even' : 'odd'))
        .toEqual({ odd: 3, even: 2 });
    });
  });

  describe('pluck()', () => {
    it('should extract values by key', () => {
      const data = [{ name: 'a' }, { name: 'b' }, { name: 'c' }];
      expect(pluck(data, 'name')).toEqual(['a', 'b', 'c']);
    });
  });

  describe('sortBy()', () => {
    it('should sort by key ascending', () => {
      const data = [{ age: 30 }, { age: 20 }, { age: 25 }];
      expect(sortBy(data, 'age').map(d => d.age)).toEqual([20, 25, 30]);
    });

    it('should sort by key descending', () => {
      const data = [{ age: 30 }, { age: 20 }, { age: 25 }];
      expect(sortBy(data, 'age', 'desc').map(d => d.age)).toEqual([30, 25, 20]);
    });
  });

  describe('partition()', () => {
    it('should partition by predicate', () => {
      const [evens, odds] = partition([1, 2, 3, 4, 5], n => n % 2 === 0);
      expect(evens).toEqual([2, 4]);
      expect(odds).toEqual([1, 3, 5]);
    });
  });

  describe('rotate()', () => {
    it('should rotate array left', () => {
      expect(rotate([1, 2, 3, 4, 5], 2)).toEqual([3, 4, 5, 1, 2]);
    });

    it('should handle negative rotation', () => {
      expect(rotate([1, 2, 3, 4, 5], -2)).toEqual([4, 5, 1, 2, 3]);
    });

    it('should handle empty array', () => {
      expect(rotate([], 2)).toEqual([]);
    });
  });
});
