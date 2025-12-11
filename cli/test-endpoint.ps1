# Test the Azure logs endpoint
$port = "40942"
$url = "http://localhost:$port/api/azure/logs"

Write-Host "Testing endpoint: $url" -ForegroundColor Cyan

try {
    $response = Invoke-RestMethod -Uri $url -Method GET -ErrorAction Stop
    Write-Host "`nSuccess! Response:" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5
} catch {
    Write-Host "`nError occurred:" -ForegroundColor Red
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__)"
    Write-Host "Message: $($_.Exception.Message)"
    
    # Try to read error response
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $errorBody = $reader.ReadToEnd()
        Write-Host "`nError Body:" -ForegroundColor Yellow
        $errorBody
    }
}
