<#
.SYNOPSIS
    Concurrent PATCH /api/orders/:id rate-limit test for Velocity.

.WHAT TO LOOK FOR
    2xx  = request passed the rate limiter and reached the handler
    429  = request was rate limited
    401  = authentication problem
    400  = invalid PATCH payload
    404  = order not found
    409  = business/order-state conflict
#>

param(
    [string]$BaseUrl = $(if ($env:VELOCITY_BASE_URL) {
        $env:VELOCITY_BASE_URL
    } else {
        "http://localhost:8080"
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

$uriBase = "$BaseUrl/api/orders"

Write-Host ""
Write-Host "Velocity Modify Order Rate-Limit Test"
Write-Host "---------------------------------------------------------------"
Write-Host "Target:   PATCH $uriBase/:id"
Write-Host "Requests: $($OrderIds.Count) (fired concurrently)"
Write-Host "---------------------------------------------------------------"
Write-Host ""

$tasks = @()

$patchMethod = [System.Net.Http.HttpMethod]::new("PATCH")

for ($i = 0; $i -lt $OrderIds.Count; $i++) {

    $orderId = $OrderIds[$i]

    # Give every request a different price.
    # This keeps the PATCH operation meaningful.
    $newPrice = 210000 + $i

    $bodyObject = @{
        price    = $newPrice
        quantity = 1
    }

    $body = $bodyObject | ConvertTo-Json -Compress

    $content = [System.Net.Http.StringContent]::new(
        $body,
        [System.Text.Encoding]::UTF8,
        "application/json"
    )

    $uri = "$uriBase/$orderId"

    $request = [System.Net.Http.HttpRequestMessage]::new(
        $patchMethod,
        $uri
    )

    $request.Content = $content

    $tasks += , $client.SendAsync($request)
}

[System.Threading.Tasks.Task]::WaitAll($tasks)

$successful = 0
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
        $successful++
    }
    else {
        $other++
    }
}

$client.Dispose()

Write-Host "---------------------------------------------------------------"
Write-Host "Successful: $successful   Rate-limited (429): $limited   Other: $other"
Write-Host "---------------------------------------------------------------"

if ($limited -eq 0) {
    Write-Warning "No PATCH request was rate-limited."
    Write-Warning "If Other > 0, inspect those responses before judging the limiter."
    exit 2
}

Write-Host ""
Write-Host "PATCH rate limiting is confirmed if 429 responses are present."