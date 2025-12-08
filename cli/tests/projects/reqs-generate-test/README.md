# Requirements Generate Test Project

This project tests the `azd app reqs --generate` command with various edge cases.

## Test Cases

1. **No azure.yaml** - Tests that azure.yaml is created with detected reqs
2. **Empty reqs array** - Tests that `reqs: []` is properly replaced
3. **Existing reqs** - Tests that new reqs are merged without duplicates

## Usage

```bash
# Test with no azure.yaml (will create one)
rm azure.yaml
azd app reqs -g

# Test with inline empty array
echo 'name: test\nreqs: []' > azure.yaml
azd app reqs -g
```
