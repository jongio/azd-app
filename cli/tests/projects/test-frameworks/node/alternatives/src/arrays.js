/**
 * Array utilities module - demonstrates various array operations
 * Shared by both Mocha and Jasmine test suites
 */

export function unique(arr) {
  return [...new Set(arr)];
}

export function flatten(arr) {
  return arr.flat(Infinity);
}

export function chunk(arr, size) {
  if (size <= 0) return [];
  const result = [];
  for (let i = 0; i < arr.length; i += size) {
    result.push(arr.slice(i, i + size));
  }
  return result;
}

export function shuffle(arr) {
  const result = [...arr];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}

export function sum(arr) {
  return arr.reduce((acc, val) => acc + val, 0);
}

export function average(arr) {
  if (arr.length === 0) return 0;
  return sum(arr) / arr.length;
}

export function min(arr) {
  if (arr.length === 0) return undefined;
  return Math.min(...arr);
}

export function max(arr) {
  if (arr.length === 0) return undefined;
  return Math.max(...arr);
}

export function compact(arr) {
  return arr.filter(Boolean);
}

export function intersection(arr1, arr2) {
  const set2 = new Set(arr2);
  return arr1.filter(x => set2.has(x));
}

export function difference(arr1, arr2) {
  const set2 = new Set(arr2);
  return arr1.filter(x => !set2.has(x));
}

export function union(arr1, arr2) {
  return [...new Set([...arr1, ...arr2])];
}

export function groupBy(arr, key) {
  return arr.reduce((groups, item) => {
    const group = item[key];
    groups[group] = groups[group] || [];
    groups[group].push(item);
    return groups;
  }, {});
}

export function countBy(arr, key) {
  return arr.reduce((counts, item) => {
    const value = typeof key === 'function' ? key(item) : item[key];
    counts[value] = (counts[value] || 0) + 1;
    return counts;
  }, {});
}

export function pluck(arr, key) {
  return arr.map(item => item[key]);
}

export function sortBy(arr, key, order = 'asc') {
  return [...arr].sort((a, b) => {
    const valA = typeof key === 'function' ? key(a) : a[key];
    const valB = typeof key === 'function' ? key(b) : b[key];
    if (order === 'desc') {
      return valB > valA ? 1 : valB < valA ? -1 : 0;
    }
    return valA > valB ? 1 : valA < valB ? -1 : 0;
  });
}

export function partition(arr, predicate) {
  const truthy = [];
  const falsy = [];
  for (const item of arr) {
    (predicate(item) ? truthy : falsy).push(item);
  }
  return [truthy, falsy];
}

export function rotate(arr, n) {
  if (arr.length === 0) return [];
  const shift = ((n % arr.length) + arr.length) % arr.length;
  return [...arr.slice(shift), ...arr.slice(0, shift)];
}
