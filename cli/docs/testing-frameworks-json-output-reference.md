# Testing Frameworks JSON Output Reference

A comprehensive reference for JSON/machine-readable output capabilities across testing frameworks in Node.js, Python, .NET, and Go. This document is designed to help build a unified parser system for test results.

---

## Table of Contents

- [Node.js Frameworks](#nodejs-frameworks)
  - [Jest](#jest)
  - [Vitest](#vitest)
  - [Mocha](#mocha)
  - [AVA](#ava)
  - [Node-TAP](#node-tap)
  - [Playwright Test](#playwright-test)
- [Python Frameworks](#python-frameworks)
  - [pytest](#pytest)
  - [unittest](#unittest)
  - [Robot Framework](#robot-framework)
- [.NET Frameworks](#net-frameworks)
  - [dotnet test (xUnit/NUnit/MSTest)](#dotnet-test)
  - [xUnit](#xunit)
  - [NUnit](#nunit)
- [Go Frameworks](#go-frameworks)
  - [go test](#go-test)
  - [Ginkgo](#ginkgo)
- [Coverage JSON Formats](#coverage-json-formats)
- [Unified Schema Considerations](#unified-schema-considerations)

---

## Node.js Frameworks

### Jest

**Website:** https://jestjs.io/

#### CLI Flags for JSON Output

```bash
# Output JSON to stdout
jest --json

# Output JSON to a file
jest --json --outputFile=results.json

# With coverage
jest --json --coverage --outputFile=results.json
```

#### Configuration (jest.config.js)

```javascript
module.exports = {
  // Custom results processor for post-processing
  testResultsProcessor: './my-processor.js',
  
  // Coverage output formats
  coverageReporters: ['json', 'json-summary', 'lcov', 'text'],
  coverageDirectory: './coverage'
};
```

#### JSON Output Schema

```json
{
  "numFailedTestSuites": 0,
  "numFailedTests": 0,
  "numPassedTestSuites": 2,
  "numPassedTests": 10,
  "numPendingTestSuites": 0,
  "numPendingTests": 0,
  "numRuntimeErrorTestSuites": 0,
  "numTotalTestSuites": 2,
  "numTotalTests": 10,
  "openHandles": [],
  "snapshot": {
    "added": 0,
    "didUpdate": false,
    "failure": false,
    "filesAdded": 0,
    "filesRemoved": 0,
    "filesRemovedList": [],
    "filesUnmatched": 0,
    "filesUpdated": 0,
    "matched": 0,
    "total": 0,
    "unchecked": 0,
    "uncheckedKeysByFile": [],
    "unmatched": 0,
    "updated": 0
  },
  "startTime": 1697737019307,
  "success": true,
  "testResults": [
    {
      "assertionResults": [
        {
          "ancestorTitles": ["describe block"],
          "duration": 5,
          "failureDetails": [],
          "failureMessages": [],
          "fullName": "describe block test name",
          "invocations": 1,
          "location": null,
          "numPassingAsserts": 1,
          "retryReasons": [],
          "status": "passed",
          "title": "test name"
        }
      ],
      "endTime": 1697737019500,
      "message": "",
      "name": "/path/to/test.spec.js",
      "startTime": 1697737019400,
      "status": "passed",
      "summary": ""
    }
  ],
  "wasInterrupted": false
}
```

**Key Fields:**
| Field | Description |
|-------|-------------|
| `success` | Overall pass/fail status |
| `numTotalTests` | Total number of tests |
| `numPassedTests` | Number of passed tests |
| `numFailedTests` | Number of failed tests |
| `testResults[].status` | Suite status: `passed`, `failed`, `pending` |
| `assertionResults[].status` | Test status: `passed`, `failed`, `pending`, `skipped`, `todo` |
| `assertionResults[].duration` | Test duration in milliseconds |
| `assertionResults[].failureMessages` | Array of error messages |

#### Coverage JSON (coverage-summary.json)

```json
{
  "total": {
    "lines": { "total": 100, "covered": 80, "skipped": 0, "pct": 80 },
    "statements": { "total": 120, "covered": 96, "skipped": 0, "pct": 80 },
    "functions": { "total": 20, "covered": 16, "skipped": 0, "pct": 80 },
    "branches": { "total": 40, "covered": 30, "skipped": 0, "pct": 75 }
  },
  "/path/to/file.js": {
    "lines": { "total": 50, "covered": 40, "skipped": 0, "pct": 80 },
    "statements": { "total": 60, "covered": 48, "skipped": 0, "pct": 80 },
    "functions": { "total": 10, "covered": 8, "skipped": 0, "pct": 80 },
    "branches": { "total": 20, "covered": 15, "skipped": 0, "pct": 75 }
  }
}
```

---

### Vitest

**Website:** https://vitest.dev/

#### CLI Flags for JSON Output

```bash
# JSON reporter to stdout
vitest --reporter=json

# JSON reporter to file
vitest --reporter=json --outputFile=results.json

# Multiple reporters
vitest --reporter=json --reporter=default --outputFile=results.json

# With coverage
vitest --coverage --coverage.reporter=json
```

#### Configuration (vitest.config.ts)

```typescript
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    reporters: ['json', 'default'],
    outputFile: {
      json: './test-results.json',
      junit: './junit-report.xml'
    },
    coverage: {
      reporter: ['json', 'json-summary', 'lcov', 'text'],
      reportsDirectory: './coverage'
    }
  }
})
```

#### JSON Output Schema (Jest-compatible)

```json
{
  "numTotalTestSuites": 4,
  "numPassedTestSuites": 2,
  "numFailedTestSuites": 1,
  "numPendingTestSuites": 1,
  "numTotalTests": 4,
  "numPassedTests": 1,
  "numFailedTests": 1,
  "numPendingTests": 1,
  "numTodoTests": 1,
  "startTime": 1697737019307,
  "success": false,
  "testResults": [
    {
      "assertionResults": [
        {
          "ancestorTitles": ["", "first test file"],
          "fullName": " first test file 2 + 2 should equal 4",
          "status": "failed",
          "title": "2 + 2 should equal 4",
          "duration": 9,
          "failureMessages": ["expected 5 to be 4 // Object.is equality"],
          "location": {
            "line": 20,
            "column": 28
          },
          "meta": {}
        }
      ],
      "startTime": 1697737019787,
      "endTime": 1697737019797,
      "status": "failed",
      "message": "",
      "name": "/root-directory/__tests__/test-file-1.test.ts"
    }
  ],
  "coverageMap": {}
}
```

**Note:** Vitest's JSON format is designed to be Jest-compatible. Since Vitest 3, the `coverageMap` field includes coverage information when coverage is enabled.

---

### Mocha

**Website:** https://mochajs.org/

#### CLI Flags for JSON Output

```bash
# JSON reporter to stdout
mocha --reporter json

# JSON reporter to file
mocha --reporter json --reporter-option output=results.json

# JSON stream (newline-delimited JSON)
mocha --reporter json-stream
```

#### Configuration (.mocharc.json)

```json
{
  "reporter": "json",
  "reporter-option": ["output=results.json"]
}
```

#### JSON Output Schema

```json
{
  "stats": {
    "suites": 2,
    "tests": 5,
    "passes": 4,
    "pending": 0,
    "failures": 1,
    "start": "2023-10-19T17:41:58.580Z",
    "end": "2023-10-19T17:41:58.600Z",
    "duration": 20
  },
  "tests": [
    {
      "title": "test name",
      "fullTitle": "Suite Name test name",
      "file": "/path/to/test.js",
      "duration": 5,
      "currentRetry": 0,
      "speed": "fast",
      "err": {}
    }
  ],
  "pending": [],
  "failures": [
    {
      "title": "failing test",
      "fullTitle": "Suite Name failing test",
      "file": "/path/to/test.js",
      "duration": 10,
      "currentRetry": 0,
      "err": {
        "message": "expected true to equal false",
        "stack": "AssertionError: expected true to equal false\n    at Context.<anonymous>...",
        "actual": "true",
        "expected": "false",
        "operator": "strictEqual"
      }
    }
  ],
  "passes": [
    {
      "title": "passing test",
      "fullTitle": "Suite Name passing test",
      "file": "/path/to/test.js",
      "duration": 3,
      "currentRetry": 0,
      "speed": "fast",
      "err": {}
    }
  ]
}
```

#### JSON Stream Format

Each line is a separate JSON object:

```json
["start",{"total":5}]
["pass",{"title":"test name","fullTitle":"Suite test name","duration":5,"currentRetry":0,"speed":"fast"}]
["fail",{"title":"failing test","fullTitle":"Suite failing test","duration":10,"err":{"message":"error"}}]
["end",{"suites":2,"tests":5,"passes":4,"pending":0,"failures":1,"start":"...","end":"...","duration":20}]
```

**Key Fields:**
| Field | Description |
|-------|-------------|
| `stats.tests` | Total number of tests |
| `stats.passes` | Number of passed tests |
| `stats.failures` | Number of failed tests |
| `stats.pending` | Number of skipped/pending tests |
| `stats.duration` | Total duration in milliseconds |
| `tests[].duration` | Individual test duration |
| `failures[].err.message` | Error message |
| `failures[].err.stack` | Stack trace |

**Notes:**
- JSON-stream reporter is **not compatible** with parallel mode
- Use `--reporter-option output=filename.json` to write to file

---

### AVA

**Website:** https://github.com/avajs/ava

#### CLI Flags for TAP Output

AVA does not have native JSON output but supports TAP format:

```bash
# TAP output (pipe to TAP-to-JSON converter)
npx ava --tap

# Pipe to TAP reporter
npx ava --tap | npx tap-json
```

#### Configuration (ava.config.js)

```javascript
export default {
  tap: true,
  verbose: true
};
```

#### TAP Output Format

```
TAP version 13
# test name
ok 1 - passing test
not ok 2 - failing test
  ---
  name: AssertionError
  message: expected true to equal false
  at: test.js:10:5
  ...
1..2
# tests 2
# pass 1
# fail 1
```

**Parsing Recommendation:** Use a TAP parser library (e.g., `tap-parser`) to convert TAP to JSON.

**Notes:**
- TAP reporter is **unavailable in watch mode**
- No native JSON reporter

---

### Node-TAP

**Website:** https://node-tap.org/

#### CLI Flags for JSON Output

```bash
# JSON reporter
tap --reporter=json

# JSON stream (newline-delimited)
tap --reporter=jsonstream

# Output to file
tap --reporter=json --reporter-file=results.json

# Coverage in JSON
tap --coverage-report=json
```

#### Configuration (.taprc or package.json)

```yaml
# .taprc
reporter: json
reporter-file: results.json
coverage-report: json
```

#### Built-in Reporters

- `base` - Default human-readable
- `json` - Full JSON output
- `jsonstream` - Newline-delimited JSON
- `junit` - JUnit XML format
- `tap` - Raw TAP output

#### Coverage Reporters

- `json` - Istanbul JSON format
- `json-summary` - Summary only
- `lcov` - LCOV format
- `html` - HTML report
- `text` - Terminal output

---

### Playwright Test

**Website:** https://playwright.dev/

#### CLI Flags for JSON Output

```bash
# JSON reporter to stdout
npx playwright test --reporter=json

# JSON reporter to file (using environment variable)
PLAYWRIGHT_JSON_OUTPUT_NAME=results.json npx playwright test --reporter=json

# PowerShell
$env:PLAYWRIGHT_JSON_OUTPUT_NAME="results.json"; npx playwright test --reporter=json
```

#### Configuration (playwright.config.ts)

```typescript
import { defineConfig } from '@playwright/test';

export default defineConfig({
  reporter: [
    ['json', { outputFile: 'results.json' }],
    ['html', { outputFolder: 'playwright-report' }],
    ['junit', { outputFile: 'junit-results.xml' }]
  ]
});
```

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `PLAYWRIGHT_JSON_OUTPUT_NAME` | Output file name |
| `PLAYWRIGHT_JSON_OUTPUT_DIR` | Output directory |
| `PLAYWRIGHT_JSON_OUTPUT_FILE` | Full path (overrides above) |

#### JSON Output Schema

```json
{
  "config": {
    "projects": [...],
    "version": "1.40.0"
  },
  "suites": [
    {
      "title": "test-file.spec.ts",
      "file": "test-file.spec.ts",
      "column": 0,
      "line": 0,
      "specs": [
        {
          "title": "test name",
          "ok": true,
          "tags": [],
          "tests": [
            {
              "timeout": 30000,
              "annotations": [],
              "expectedStatus": "passed",
              "projectId": "chromium",
              "projectName": "chromium",
              "results": [
                {
                  "workerIndex": 0,
                  "status": "passed",
                  "duration": 1234,
                  "errors": [],
                  "stdout": [],
                  "stderr": [],
                  "retry": 0,
                  "startTime": "2023-10-19T17:41:58.580Z",
                  "attachments": []
                }
              ],
              "status": "expected"
            }
          ]
        }
      ],
      "suites": []
    }
  ],
  "errors": [],
  "stats": {
    "startTime": "2023-10-19T17:41:58.000Z",
    "duration": 5000,
    "expected": 10,
    "unexpected": 0,
    "flaky": 0,
    "skipped": 0
  }
}
```

---

## Python Frameworks

### pytest

**Website:** https://docs.pytest.org/

#### CLI Flags for Machine-Readable Output

```bash
# JUnit XML output (built-in)
pytest --junitxml=results.xml

# JSON output (requires plugin)
pytest --json-report --json-report-file=results.json

# Coverage JSON
pytest --cov=myproject --cov-report=json
```

#### Required Plugins for JSON

```bash
# Install JSON report plugin
pip install pytest-json-report

# Install coverage plugin
pip install pytest-cov
```

#### pytest-json-report Configuration (pytest.ini)

```ini
[pytest]
json_report_file = results.json
json_report_indent = 2
json_report_omit = collectors
```

#### JSON Output Schema (pytest-json-report)

```json
{
  "created": 1697737019.123,
  "duration": 1.234,
  "exitcode": 0,
  "root": "/path/to/project",
  "environment": {
    "Python": "3.11.0",
    "Platform": "Linux-5.4.0",
    "pytest": "7.4.0"
  },
  "summary": {
    "passed": 10,
    "failed": 2,
    "error": 0,
    "skipped": 1,
    "xfailed": 0,
    "xpassed": 0,
    "total": 13,
    "collected": 13
  },
  "tests": [
    {
      "nodeid": "tests/test_example.py::test_function",
      "lineno": 10,
      "keywords": ["test_function", "test_example.py", "tests"],
      "outcome": "passed",
      "setup": { "duration": 0.001, "outcome": "passed" },
      "call": { "duration": 0.123, "outcome": "passed" },
      "teardown": { "duration": 0.001, "outcome": "passed" }
    },
    {
      "nodeid": "tests/test_example.py::test_failing",
      "lineno": 20,
      "outcome": "failed",
      "call": {
        "duration": 0.050,
        "outcome": "failed",
        "crash": {
          "path": "/path/to/tests/test_example.py",
          "lineno": 25,
          "message": "assert 1 == 2"
        },
        "traceback": [
          {
            "path": "tests/test_example.py",
            "lineno": 25,
            "message": "AssertionError"
          }
        ],
        "longrepr": "..."
      }
    }
  ],
  "collectors": [],
  "warnings": []
}
```

**Key Fields:**
| Field | Description |
|-------|-------------|
| `summary.passed` | Number of passed tests |
| `summary.failed` | Number of failed tests |
| `summary.total` | Total number of tests |
| `tests[].outcome` | Test outcome: `passed`, `failed`, `skipped`, `xfailed`, `xpassed` |
| `tests[].call.duration` | Test execution duration |
| `tests[].call.crash` | Error location information |

#### Coverage JSON (coverage.json via pytest-cov)

Uses Istanbul/NYC compatible format - see Coverage JSON Formats section.

---

### unittest

**Website:** https://docs.python.org/3/library/unittest.html

Python's built-in unittest does **not** have native JSON output.

#### Options for JSON Output

1. **Use pytest to run unittest tests:**
   ```bash
   pytest --json-report tests/
   ```

2. **Use xmlrunner for JUnit XML:**
   ```bash
   pip install xmlrunner
   ```
   
   ```python
   import xmlrunner
   unittest.main(testRunner=xmlrunner.XMLTestRunner(output='test-reports'))
   ```

3. **Custom TestResult subclass:**
   ```python
   import json
   import unittest
   
   class JsonTestResult(unittest.TestResult):
       def __init__(self):
           super().__init__()
           self.results = []
       
       def addSuccess(self, test):
           self.results.append({
               'name': str(test),
               'status': 'passed'
           })
       
       def addFailure(self, test, err):
           self.results.append({
               'name': str(test),
               'status': 'failed',
               'message': str(err[1])
           })
       
       def to_json(self):
           return json.dumps({
               'tests': len(self.results),
               'failures': len(self.failures),
               'errors': len(self.errors),
               'results': self.results
           })
   ```

---

### Robot Framework

**Website:** https://robotframework.org/

#### CLI Flags for JSON Output

```bash
# JSON output (Robot Framework 7.0+)
robot --output output.json tests/

# Also supports XML output
robot --output output.xml tests/

# Log in JSON format
robot --log NONE --report NONE --output output.json tests/
```

#### JSON Output Schema (result.json)

```json
{
  "generator": "Robot 7.0",
  "generated": "2023-10-19T17:41:58.000Z",
  "rpa": false,
  "suite": {
    "name": "Tests",
    "doc": "",
    "source": "/path/to/tests",
    "suites": [],
    "tests": [
      {
        "name": "Test Case Name",
        "doc": "Test documentation",
        "tags": ["smoke", "critical"],
        "timeout": null,
        "status": {
          "status": "PASS",
          "starttime": "2023-10-19T17:41:58.000Z",
          "endtime": "2023-10-19T17:41:59.000Z",
          "elapsed": 1.234
        },
        "keywords": [...]
      }
    ],
    "status": {
      "status": "PASS",
      "starttime": "...",
      "endtime": "...",
      "elapsed": 5.678
    }
  },
  "statistics": {
    "total": {
      "pass": 10,
      "fail": 2,
      "skip": 1
    },
    "tag": {...},
    "suite": {...}
  },
  "errors": []
}
```

---

## .NET Frameworks

### dotnet test

**Applies to:** xUnit, NUnit, MSTest

#### CLI Flags for Machine-Readable Output

```bash
# TRX format (Visual Studio Test Results)
dotnet test --logger "trx;LogFileName=results.trx"

# JUnit format (requires logger package)
dotnet test --logger "junit;LogFileName=results.xml"

# Console logger with verbosity
dotnet test --logger "console;verbosity=detailed"

# Multiple loggers
dotnet test --logger trx --logger "console;verbosity=normal"

# JSON output (limited - build output only)
dotnet test -json
```

#### Required Packages for Additional Formats

```xml
<!-- For JUnit XML output -->
<PackageReference Include="JunitXml.TestLogger" Version="3.0.124" />

<!-- For JSON output -->
<PackageReference Include="Meziantou.Extensions.Logging.InMemory" Version="1.0.0" />
```

#### TRX Format (XML)

The TRX format is Visual Studio's native test result format:

```xml
<?xml version="1.0" encoding="utf-8"?>
<TestRun id="..." name="..." runUser="...">
  <Results>
    <UnitTestResult testId="..." testName="TestMethod" 
                    outcome="Passed" duration="00:00:00.123">
      <Output>
        <StdOut>...</StdOut>
      </Output>
    </UnitTestResult>
  </Results>
  <ResultSummary outcome="Completed">
    <Counters total="10" passed="8" failed="2" />
  </ResultSummary>
</TestRun>
```

---

### xUnit

**Website:** https://xunit.net/

xUnit outputs results in xUnit v2 XML format by default when using `dotnet test`.

#### XML v2 Format Schema

```xml
<?xml version="1.0" encoding="utf-8"?>
<assemblies timestamp="2023-10-19T17:41:58.000Z">
  <assembly name="MyTests" 
            run-date="2023-10-19" 
            run-time="17:41:58"
            total="10" 
            passed="8" 
            failed="2" 
            skipped="0"
            time="1.234"
            errors="0">
    <collection name="Test Collection" 
                total="5" 
                passed="4" 
                failed="1" 
                skipped="0" 
                time="0.567">
      <test name="Namespace.Class.Method" 
            type="Namespace.Class" 
            method="Method"
            time="0.123" 
            result="Pass">
        <traits>
          <trait name="Category" value="Unit" />
        </traits>
      </test>
      <test name="Namespace.Class.FailingMethod"
            type="Namespace.Class"
            method="FailingMethod"
            time="0.050"
            result="Fail">
        <failure exception-type="Xunit.Sdk.EqualException">
          <message>Assert.Equal() Failure</message>
          <stack-trace>at Namespace.Class.FailingMethod()...</stack-trace>
        </failure>
      </test>
    </collection>
  </assembly>
</assemblies>
```

---

### NUnit

**Website:** https://nunit.org/

#### CLI Flags (NUnit Console Runner)

```bash
# NUnit3 format (default)
nunit3-console tests.dll --result=results.xml

# NUnit2 format
nunit3-console tests.dll --result=results.xml;format=nunit2

# Multiple result files
nunit3-console tests.dll --result=nunit3.xml --result=nunit2.xml;format=nunit2
```

#### NUnit3 XML Format

```xml
<?xml version="1.0" encoding="utf-8"?>
<test-run id="0" 
          testcasecount="10" 
          result="Failed" 
          total="10" 
          passed="8" 
          failed="2" 
          inconclusive="0" 
          skipped="0"
          start-time="2023-10-19T17:41:58.000Z" 
          end-time="2023-10-19T17:41:59.234Z"
          duration="1.234">
  <test-suite type="Assembly" name="MyTests.dll" fullname="MyTests.dll">
    <test-suite type="TestFixture" name="TestClass" fullname="Namespace.TestClass">
      <test-case id="1" 
                 name="TestMethod" 
                 fullname="Namespace.TestClass.TestMethod"
                 result="Passed" 
                 duration="0.123"
                 asserts="1">
      </test-case>
      <test-case id="2" 
                 name="FailingTest" 
                 fullname="Namespace.TestClass.FailingTest"
                 result="Failed" 
                 duration="0.050"
                 asserts="1">
        <failure>
          <message>Expected: 1 But was: 2</message>
          <stack-trace>at Namespace.TestClass.FailingTest()...</stack-trace>
        </failure>
      </test-case>
    </test-suite>
  </test-suite>
</test-run>
```

---

## Go Frameworks

### go test

**Website:** https://golang.org/cmd/go/

#### CLI Flags for JSON Output

```bash
# JSON output
go test -json ./...

# JSON output to file
go test -json ./... > results.json

# With verbose output
go test -v -json ./...

# With coverage
go test -json -cover -coverprofile=coverage.out ./...

# Convert coverage to JSON
go tool cover -func=coverage.out
```

#### JSON Output Format (test2json)

Each line is a separate JSON event:

```json
{"Time":"2023-10-19T17:41:58.000Z","Action":"start","Package":"mypackage"}
{"Time":"2023-10-19T17:41:58.001Z","Action":"run","Package":"mypackage","Test":"TestFunction"}
{"Time":"2023-10-19T17:41:58.100Z","Action":"output","Package":"mypackage","Test":"TestFunction","Output":"=== RUN   TestFunction\n"}
{"Time":"2023-10-19T17:41:58.200Z","Action":"output","Package":"mypackage","Test":"TestFunction","Output":"--- PASS: TestFunction (0.10s)\n"}
{"Time":"2023-10-19T17:41:58.200Z","Action":"pass","Package":"mypackage","Test":"TestFunction","Elapsed":0.1}
{"Time":"2023-10-19T17:41:58.300Z","Action":"output","Package":"mypackage","Output":"PASS\n"}
{"Time":"2023-10-19T17:41:58.300Z","Action":"pass","Package":"mypackage","Elapsed":0.3}
```

**Event Actions:**
| Action | Description |
|--------|-------------|
| `start` | Package test started |
| `run` | Individual test started |
| `pause` | Test paused (parallel) |
| `cont` | Test continued |
| `output` | Test output line |
| `bench` | Benchmark result |
| `pass` | Test/package passed |
| `fail` | Test/package failed |
| `skip` | Test skipped |

**JSON Fields:**
| Field | Description |
|-------|-------------|
| `Time` | Event timestamp (RFC3339) |
| `Action` | Event type |
| `Package` | Package being tested |
| `Test` | Test name (if applicable) |
| `Output` | Output text (for output action) |
| `Elapsed` | Duration in seconds (for pass/fail) |

---

### Ginkgo

**Website:** https://onsi.github.io/ginkgo/

#### CLI Flags for JSON Output

```bash
# JSON report
ginkgo --json-report=report.json ./...

# JUnit report
ginkgo --junit-report=report.xml ./...

# Go test JSON format
ginkgo --gojson-report=report.json ./...

# Multiple reports
ginkgo --json-report=ginkgo.json --junit-report=junit.xml ./...
```

#### JSON Report Schema (--json-report)

```json
[
  {
    "SuitePath": "/path/to/suite",
    "SuiteDescription": "My Test Suite",
    "SuiteSucceeded": false,
    "PreRunStats": {
      "TotalSpecs": 10,
      "SpecsThatWillRun": 10
    },
    "RunTime": 1234567890,
    "SpecReports": [
      {
        "ContainerHierarchyTexts": ["Describe Block"],
        "ContainerHierarchyLocations": [...],
        "LeafNodeType": "It",
        "LeafNodeLocation": {
          "FileName": "/path/to/test.go",
          "LineNumber": 25
        },
        "LeafNodeText": "should do something",
        "State": "passed",
        "StartTime": "2023-10-19T17:41:58.000Z",
        "EndTime": "2023-10-19T17:41:58.100Z",
        "RunTime": 100000000,
        "NumAttempts": 1,
        "CapturedGinkgoWriterOutput": "",
        "CapturedStdOutErr": "",
        "Failure": null
      },
      {
        "LeafNodeText": "should fail",
        "State": "failed",
        "Failure": {
          "Message": "Expected true to be false",
          "Location": {
            "FileName": "/path/to/test.go",
            "LineNumber": 30
          },
          "FailureNodeType": "It"
        }
      }
    ],
    "SuiteHasProgrammaticFocus": false,
    "SpecialSuiteFailureReasons": []
  }
]
```

**Spec States:**
- `passed` - Test passed
- `failed` - Test failed
- `pending` - Test marked pending
- `skipped` - Test skipped
- `panicked` - Test panicked
- `interrupted` - Test interrupted
- `aborted` - Suite aborted

---

## Coverage JSON Formats

### Istanbul/NYC Format (Used by Jest, Vitest, Node-TAP, pytest-cov)

```json
{
  "/path/to/file.js": {
    "path": "/path/to/file.js",
    "statementMap": {
      "0": { "start": { "line": 1, "column": 0 }, "end": { "line": 1, "column": 20 } }
    },
    "fnMap": {
      "0": { "name": "functionName", "decl": {...}, "loc": {...}, "line": 5 }
    },
    "branchMap": {
      "0": { "type": "if", "locations": [...], "line": 10 }
    },
    "s": { "0": 5, "1": 3 },
    "f": { "0": 2 },
    "b": { "0": [3, 2] }
  }
}
```

**Summary Format (coverage-summary.json):**

```json
{
  "total": {
    "lines": { "total": 100, "covered": 80, "skipped": 0, "pct": 80 },
    "statements": { "total": 120, "covered": 96, "skipped": 0, "pct": 80 },
    "functions": { "total": 20, "covered": 16, "skipped": 0, "pct": 80 },
    "branches": { "total": 40, "covered": 30, "skipped": 0, "pct": 75 },
    "branchesTrue": { "total": 0, "covered": 0, "skipped": 0, "pct": 100 }
  }
}
```

### Go Cover Profile Format

```
mode: set
mypackage/file.go:10.2,12.16 1 1
mypackage/file.go:12.16,14.3 2 1
mypackage/file.go:16.2,18.16 1 0
```

Format: `filename:startLine.startCol,endLine.endCol statements count`

---

## Unified Schema Considerations

When building a parser system, consider normalizing to a common schema:

```typescript
interface TestResults {
  framework: string;
  version?: string;
  timestamp: string;
  duration: number;
  
  summary: {
    total: number;
    passed: number;
    failed: number;
    skipped: number;
    pending: number;
    errors: number;
  };
  
  suites: TestSuite[];
  
  coverage?: {
    lines: CoverageMetric;
    statements: CoverageMetric;
    branches: CoverageMetric;
    functions: CoverageMetric;
  };
}

interface TestSuite {
  name: string;
  file?: string;
  duration: number;
  tests: TestCase[];
  suites?: TestSuite[];
}

interface TestCase {
  name: string;
  fullName: string;
  status: 'passed' | 'failed' | 'skipped' | 'pending' | 'error';
  duration: number;
  file?: string;
  line?: number;
  error?: {
    message: string;
    stack?: string;
    expected?: string;
    actual?: string;
  };
  retries?: number;
}

interface CoverageMetric {
  total: number;
  covered: number;
  percentage: number;
}
```

---

## Quick Reference Table

| Framework | JSON Flag | Output File Flag | Coverage JSON |
|-----------|-----------|------------------|---------------|
| **Jest** | `--json` | `--outputFile=<file>` | `--coverage` |
| **Vitest** | `--reporter=json` | `--outputFile=<file>` | `--coverage` |
| **Mocha** | `--reporter json` | `--reporter-option output=<file>` | Use nyc |
| **AVA** | `--tap` (TAP only) | Pipe to file | Use c8/nyc |
| **Node-TAP** | `--reporter=json` | `--reporter-file=<file>` | `--coverage-report=json` |
| **Playwright** | `--reporter=json` | Config or `PLAYWRIGHT_JSON_OUTPUT_NAME` | N/A |
| **pytest** | Plugin required | `--json-report-file=<file>` | `--cov-report=json` |
| **Robot** | `--output <file>.json` | Same flag | N/A |
| **dotnet test** | `--logger trx` (XML) | `--logger "trx;LogFileName=<file>"` | `--collect:"XPlat Code Coverage"` |
| **go test** | `-json` | Redirect stdout | `-coverprofile=<file>` |
| **Ginkgo** | `--json-report=<file>` | Same flag | Use go cover |

---

## Notes

1. **Jest and Vitest** share the same JSON schema, making them interchangeable for parsing
2. **Mocha's JSON-stream** is not compatible with parallel mode
3. **AVA** requires TAP output plus a TAP-to-JSON converter
4. **Python unittest** has no native JSON; use pytest or custom TestResult
5. **.NET** primarily uses TRX (XML) format; JSON requires additional packages
6. **Go test** outputs newline-delimited JSON events, not a single JSON object
7. **Coverage formats** are largely standardized around Istanbul/NYC format for JavaScript
