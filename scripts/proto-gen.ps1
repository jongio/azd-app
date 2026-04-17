# proto-gen.ps1 - Generate Go + TypeScript code from .proto files.
#
# Run from the repo root:
#   pwsh -NoProfile -File scripts/proto-gen.ps1
#
# Requires (Go side, install via `go install`):
#   github.com/bufbuild/buf/cmd/buf@v1.68.2
#   google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.x
#   connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.x
#
# Requires (TS side, installed via pnpm in cli/dashboard):
#   @bufbuild/protoc-gen-es
#   @connectrpc/protoc-gen-connect-es

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$protoDir = Join-Path $repoRoot 'proto'
$goBin    = Join-Path (& go env GOPATH) 'bin'
$tsBin    = Join-Path $repoRoot 'cli\dashboard\node_modules\.bin'

if (-not (Test-Path $goBin))  { throw "Go bin directory not found: $goBin" }
if (-not (Test-Path $tsBin))  { throw "TS bin directory not found: $tsBin (run pnpm install in cli/dashboard)" }

# Prepend tool dirs so buf finds the plugin binaries on PATH.
$env:PATH = "$goBin;$tsBin;$env:PATH"

Write-Host "Running buf generate from $protoDir" -ForegroundColor Cyan
Push-Location $protoDir
try {
    & buf lint
    if ($LASTEXITCODE -ne 0) { throw "buf lint failed" }

    & buf build
    if ($LASTEXITCODE -ne 0) { throw "buf build failed" }

    Push-Location $repoRoot
    try {
        & buf generate proto
        if ($LASTEXITCODE -ne 0) { throw "buf generate failed" }
    }
    finally {
        Pop-Location
    }
}
finally {
    Pop-Location
}

Write-Host "Generated code:" -ForegroundColor Green
Get-ChildItem -Recurse -File (Join-Path $repoRoot 'cli\src\gen\proto') | ForEach-Object {
    Write-Host ("  go: {0}" -f $_.FullName.Substring($repoRoot.Path.Length + 1))
}
Get-ChildItem -Recurse -File (Join-Path $repoRoot 'cli\dashboard\src\gen\proto') | ForEach-Object {
    Write-Host ("  ts: {0}" -f $_.FullName.Substring($repoRoot.Path.Length + 1))
}
