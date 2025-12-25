# Test script to call MCP server
$getServicesRequest = @'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_services","arguments":{}}}
'@

$getLogsRequest = @'
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_service_logs","arguments":{"serviceName":"api","tail":5}}}
'@

Write-Host "Testing get_services..." -ForegroundColor Cyan
$getServicesRequest | azd app mcp serve 2>&1

Write-Host "`nTesting get_service_logs..." -ForegroundColor Cyan  
$getLogsRequest | azd app mcp serve 2>&1
