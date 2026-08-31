<#
.SYNOPSIS
    Hammers DELETE /api/orders/:id on a running Velocity instance to verify
    rate limiting is enforced on authenticated cancel requests.

.USAGE
    $env:VELOCITY_TOKEN = "<jwt from identity-service>"

    .\loadtest_cancel_rate_limit.ps1 `
        -OrderIds @(123,456,789)

    Or:
    .\loadtest_cancel_rate_limit.ps1 `
        -OrderIds @(123,456,789) `
        -BaseUrl "http://localhost:8080"

.WHAT TO LOOK FOR
    - Valid OPEN orders should return the normal cancel success response.
    - Once the rate-limit bucket is exhausted, requests should return 429.
    - 429 responses should contain Retry-After.
    - Requests must use different OPEN order IDs so business-level
      "already cancelled/not found" errors do not interfere with the test.
#>

param(
    [string]$BaseUrl = $(if ($env:VELOCITY_BASE_URL) {
        $env:VELOCITY_BASE_URL
    } else {
        "http://localhost:3000"
    }),

    [string]$Token = $env:VELOCITY_TOKEN,

    [Parameter(Mandatory = $true)]
    [long[]]$OrderIds
)

if ([string]::IsNullOrWhiteSpace($Token)) {
    Write-Error "Set `$env:VELOCITY_TOKEN to a valid JWT before running this."
    exit 1
}

if ($OrderIds.Count -eq 0) {
    Write-Error "Provide at least one order ID using -OrderIds."
    exit 1
}

Add-Type -AssemblyName System.Net.Http

$client = [System.Net.Http.HttpClient]::new()

$client.DefaultRequestHeaders.Authorization =
    [System.Net.Http.Headers.AuthenticationHeaderValue]::new(
        "Bearer",
        $Token
    )

Write-Host "Target:   DELETE $BaseUrl/api/orders/:id"
Write-Host "Requests: $($OrderIds.Count) (fired concurrently)"
Write-Host "---------------------------------------------------------------"

$tasks = @()

for ($i = 0; $i -lt $OrderIds.Count; $i++) {

    $orderId = $OrderIds[$i]
    $uri = "$BaseUrl/api/orders/$orderId"

    $tasks += , $client.DeleteAsync($uri)
}

[System.Threading.Tasks.Task]::WaitAll($tasks)

$allowed = 0
$limited = 0
$other = 0

"{0,-5} {1,-6} {2,-18} {3,-12} {4}" -f `
    "#",
    "HTTP",
    "X-RateLimit-Rem",
    "Retry-After",
    "Order ID" |
    Write-Host

for ($i = 0; $i -lt $tasks.Count; $i++) {

    $resp = $tasks[$i].Result
    $status = [int]$resp.StatusCode
    $orderId = $OrderIds[$i]

    $remaining = "-"
    $retryAfter = "-"

    $remVals = $null
    if ($resp.Headers.TryGetValues(
        "X-RateLimit-Remaining",
        [ref]$remVals
    )) {
        $remaining = $remVals -join ""
    }

    $retryVals = $null
    if ($resp.Headers.TryGetValues(
        "Retry-After",
        [ref]$retryVals
    )) {
        $retryAfter = $retryVals -join ""
    }

    "{0,-5} {1,-6} {2,-18} {3,-12} {4}" -f `
        ($i + 1),
        $status,
        $remaining,
        $retryAfter,
        $orderId |
        Write-Host

    if ($status -eq 429) {
        $limited++
    }
    elseif ($status -ge 200 -and $status -lt 300) {
        $allowed++
    }
    else {
        $other++
    }
}

$client.Dispose()

Write-Host "---------------------------------------------------------------"
Write-Host "Successful: $allowed   Rate-limited (429): $limited   Other: $other"

if ($limited -eq 0) {
    Write-Warning "No DELETE request was rate-limited."
    Write-Warning "Increase the number of concurrent requests or verify the DELETE limiter."
    exit 2
}