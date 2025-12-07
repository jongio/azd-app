// Intentionally failing tests to verify error handling

describe('Failing Tests', () => {
  test('this test should fail - assertion error', () => {
    expect(1 + 1).toBe(3); // Wrong expectation
  });

  test('this test should pass', () => {
    expect(2 + 2).toBe(4);
  });

  test('this test should fail - string mismatch', () => {
    expect('hello').toBe('world');
  });
});
