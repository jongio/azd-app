# Test Tool for PATH Resolution Testing

This is a simple Go application designed to test the `azd app reqs --fix` functionality without requiring you to uninstall/reinstall real tools.

## Quick Setup & Test Instructions

### Step 1: Build the Test Tool

```powershell
cd C:\code\azd-app\cli\tests\test-tool
go build -o test-tool.exe test-tool.go
```

This creates `test-tool.exe` in the current directory.

### Step 2: Create an Isolated Installation Directory

```powershell
# Create a custom install location (simulating Program Files install)
mkdir C:\CustomTools\test-tool

# Move the executable there
Move-Item test-tool.exe C:\CustomTools\test-tool\
```

### Step 3: Verify It's NOT in Your PATH

```powershell
# This should fail
test-tool --version
# Expected: 'test-tool' is not recognized as an internal or external command
```

### Step 4: Create Test Project with azure.yaml

```powershell
# Create test project directory
mkdir C:\temp\path-fix-test
cd C:\temp\path-fix-test

# Create azure.yaml requiring test-tool
@"
name: path-fix-test
reqs:
  - name: test-tool
    minVersion: 2.0.0
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
"@ | Out-File -Encoding utf8 azure.yaml
```

### Step 5: Run Initial Check (Should Fail)

```powershell
azd app reqs
```

**Expected output:**
```
✓ Checking prerequisites

✗ test-tool: NOT INSTALLED (required: 2.0.0)

💡 If you recently installed any missing tools, run 'azd app reqs --fix' to refresh PATH

✗ Some prerequisites are not satisfied
```

### Step 6: Add to PATH (Without Restarting Terminal)

```powershell
# Add to User PATH via Registry
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
[Environment]::SetEnvironmentVariable('Path', "$userPath;C:\CustomTools\test-tool", 'User')

# Verify it's NOT in current session PATH
$env:PATH
# Should NOT contain C:\CustomTools\test-tool

# Verify the command still fails in current session
test-tool --version
# Still fails because current session PATH not updated
```

### Step 7: Run Fix (Should Succeed!)

```powershell
azd app reqs --fix
```

**Expected output:**
```
🔧 Attempting to fix requirement issues...
   ✗ test-tool: NOT INSTALLED (required: 2.0.0)

🔄 Refreshing environment PATH...
   ✓ PATH refreshed successfully

🔍 Searching for test-tool...
   ✓ Found: C:\CustomTools\test-tool\test-tool.exe
   ✓ Version verified successfully

✓ Fixed 1 of 1 issues!

✓ All prerequisites satisfied after fix!
```

### Step 8: Verify Fix Worked

```powershell
# Run check again (cache was cleared by --fix, so this gets fresh results)
azd app reqs

# Expected: All satisfied
# ✓ test-tool: 2.5.0 (required: 2.0.0)
# ✓ All reqs satisfied!
```

## Test Scenarios

### Scenario A: Version Mismatch

```yaml
# Require newer version than installed
reqs:
  - name: test-tool
    minVersion: 3.0.0  # Installed is 2.5.0
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
```

**Expected:** Fix finds tool but reports version mismatch with upgrade suggestion.

### Scenario B: Tool Not Installed at All

```yaml
# Require non-existent tool
reqs:
  - name: fake-tool-xyz
    minVersion: 1.0.0
```

**Expected:** Fix cannot find tool, provides generic install suggestion.

### Scenario C: Multiple Tools (Mixed Success)

```yaml
reqs:
  - name: test-tool
    minVersion: 2.0.0
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
  - name: go
    minVersion: 1.20.0
  - name: nonexistent-xyz
    minVersion: 1.0.0
```

**Expected:** Fixes test-tool, go already satisfied, nonexistent-xyz still missing.

## Cleanup

```powershell
# Remove from PATH
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$newPath = $userPath -replace ';C:\\CustomTools\\test-tool', ''
[Environment]::SetEnvironmentVariable('Path', $newPath, 'User')

# Delete custom tools directory
Remove-Item -Recurse -Force C:\CustomTools

# Delete test project
Remove-Item -Recurse -Force C:\temp\path-fix-test
```

## Advanced Testing: Custom Version Checks

You can create multiple versions to test version comparison:

```powershell
# Build version 1.0.0
$content = (Get-Content test-tool.go) -replace 'const version = "2.5.0"', 'const version = "1.0.0"'
$content | Set-Content test-tool-old.go
go build -o C:\CustomTools\test-tool-old\test-tool.exe test-tool-old.go

# Now you can test version mismatch scenarios
```

## JSON Output Testing

```powershell
# Test with JSON output
azd app reqs --fix --output json | ConvertFrom-Json | ConvertTo-Json -Depth 10

# Validate JSON structure has expected fields:
# - success
# - fixed
# - total
# - allSatisfied
# - fixes[]
# - results[]
```

## Why This Works Better Than Uninstalling Real Tools

1. **Fast**: Build in seconds, no large installers
2. **Isolated**: Doesn't affect your real development environment
3. **Repeatable**: Can delete and recreate instantly
4. **Controllable**: Easy to modify version, command behavior
5. **Safe**: No risk to production tools
6. **Multiple Scenarios**: Can create multiple versions/instances
