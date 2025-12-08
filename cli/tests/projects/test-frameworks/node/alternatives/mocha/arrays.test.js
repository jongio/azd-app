import { expect } from 'chai';
import {
  unique, flatten, chunk, sum, average, min, max,
  compact, intersection, difference, union,
  groupBy, countBy, pluck, sortBy, partition, rotate
} from '../src/arrays.js';

describe('Array Utilities - Mocha Tests', function() {
  describe('unique()', function() {
    it('should remove duplicate values', function() {
      expect(unique([1, 2, 2, 3, 3, 3])).to.deep.equal([1, 2, 3]);
    });

    it('should handle empty arrays', function() {
      expect(unique([])).to.deep.equal([]);
    });

    it('should preserve order', function() {
      expect(unique([3, 1, 2, 1, 3])).to.deep.equal([3, 1, 2]);
    });
  });

  describe('flatten()', function() {
    it('should flatten nested arrays', function() {
      expect(flatten([1, [2, [3, [4]]]])).to.deep.equal([1, 2, 3, 4]);
    });

    it('should handle already flat arrays', function() {
      expect(flatten([1, 2, 3])).to.deep.equal([1, 2, 3]);
    });
  });

  describe('chunk()', function() {
    it('should split array into chunks', function() {
      expect(chunk([1, 2, 3, 4, 5], 2)).to.deep.equal([[1, 2], [3, 4], [5]]);
    });

    it('should handle size larger than array', function() {
      expect(chunk([1, 2], 5)).to.deep.equal([[1, 2]]);
    });

    it('should return empty array for invalid size', function() {
      expect(chunk([1, 2, 3], 0)).to.deep.equal([]);
    });
  });

  describe('sum()', function() {
    it('should sum all numbers', function() {
      expect(sum([1, 2, 3, 4])).to.equal(10);
    });

    it('should return 0 for empty array', function() {
      expect(sum([])).to.equal(0);
    });
  });

  describe('average()', function() {
    it('should calculate average', function() {
      expect(average([1, 2, 3, 4, 5])).to.equal(3);
    });

    it('should return 0 for empty array', function() {
      expect(average([])).to.equal(0);
    });
  });

  describe('min() and max()', function() {
    it('should find minimum value', function() {
      expect(min([3, 1, 4, 1, 5])).to.equal(1);
    });

    it('should find maximum value', function() {
      expect(max([3, 1, 4, 1, 5])).to.equal(5);
    });

    it('should return undefined for empty array', function() {
      expect(min([])).to.be.undefined;
      expect(max([])).to.be.undefined;
    });
  });

  describe('compact()', function() {
    it('should remove falsy values', function() {
      expect(compact([0, 1, false, 2, '', 3, null, undefined])).to.deep.equal([1, 2, 3]);
    });
  });

  describe('set operations', function() {
    it('should find intersection', function() {
      expect(intersection([1, 2, 3], [2, 3, 4])).to.deep.equal([2, 3]);
    });

    it('should find difference', function() {
      expect(difference([1, 2, 3], [2, 3, 4])).to.deep.equal([1]);
    });

    it('should find union', function() {
      expect(union([1, 2], [2, 3])).to.deep.equal([1, 2, 3]);
    });
  });

  describe('groupBy()', function() {
    it('should group by key', function() {
      const data = [
        { type: 'a', value: 1 },
        { type: 'b', value: 2 },
        { type: 'a', value: 3 }
      ];
      const result = groupBy(data, 'type');
      expect(result.a).to.have.lengthOf(2);
      expect(result.b).to.have.lengthOf(1);
    });
  });

  describe('countBy()', function() {
    it('should count by key', function() {
      const data = [{ status: 'active' }, { status: 'inactive' }, { status: 'active' }];
      expect(countBy(data, 'status')).to.deep.equal({ active: 2, inactive: 1 });
    });

    it('should count by function', function() {
      expect(countBy([1, 2, 3, 4, 5], n => n % 2 === 0 ? 'even' : 'odd'))
        .to.deep.equal({ odd: 3, even: 2 });
    });
  });

  describe('pluck()', function() {
    it('should extract values by key', function() {
      const data = [{ name: 'a' }, { name: 'b' }, { name: 'c' }];
      expect(pluck(data, 'name')).to.deep.equal(['a', 'b', 'c']);
    });
  });

  describe('sortBy()', function() {
    it('should sort by key ascending', function() {
      const data = [{ age: 30 }, { age: 20 }, { age: 25 }];
      expect(sortBy(data, 'age').map(d => d.age)).to.deep.equal([20, 25, 30]);
    });

    it('should sort by key descending', function() {
      const data = [{ age: 30 }, { age: 20 }, { age: 25 }];
      expect(sortBy(data, 'age', 'desc').map(d => d.age)).to.deep.equal([30, 25, 20]);
    });
  });

  describe('partition()', function() {
    it('should partition by predicate', function() {
      const [evens, odds] = partition([1, 2, 3, 4, 5], n => n % 2 === 0);
      expect(evens).to.deep.equal([2, 4]);
      expect(odds).to.deep.equal([1, 3, 5]);
    });
  });

  describe('rotate()', function() {
    it('should rotate array left', function() {
      expect(rotate([1, 2, 3, 4, 5], 2)).to.deep.equal([3, 4, 5, 1, 2]);
    });

    it('should handle negative rotation', function() {
      expect(rotate([1, 2, 3, 4, 5], -2)).to.deep.equal([4, 5, 1, 2, 3]);
    });

    it('should handle empty array', function() {
      expect(rotate([], 2)).to.deep.equal([]);
    });
  });
});
