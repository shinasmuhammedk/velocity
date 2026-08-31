<#
.SYNOPSIS
    Hammers POST /api/orders on a running Velocity instance to verify
    rate limiting is enforced end-to-end (config + Redis + middleware).

.USAGE
    $env:VELOCITY_TOKEN = "<jwt from identity-service>"
    .\loadtest_rate_limit.ps1 -Count 40
    .\loadtest_rate_limit.ps1 -Count 40 -BaseUrl "http://localhost:3000"

.WHAT TO LOOK FOR
    - The first N requests (N = submit_burst in the active config) should
      return 201 (or a handler validation error like 400 — anything that
      isn't 429).
    - Requests after that should start returning 429 with a Retry-After
      header.
    - If EVERY request returns non-429 no matter the count, rate limiting
      is not enforced on this environment.
#>

param(
    [string]$BaseUrl = $(if ($env:VELOCITY_BASE_URL) { $env:VELOCITY_BASE_URL } else { "http://localhost:3000" }),
    [string]$Token   = $env:VELOCITY_TOKEN,
    [int]$Count      = 30
)

if ([string]::IsNullOrWhiteSpace($Token)) {
    Write-Error "Set `$env:VELOCITY_TOKEN to a valid JWT (from identity-service) before running this."
    exit 1
}

$uri = "$BaseUrl/api/orders"

# Payload matches internal/transport/http/dto/request/submit_order_request.go —
# Price/Quantity are int64 (fixed-point integers, not strings), Side/Type/
# TimeInForce are uppercase enums (see pkg/constants/order.go).
$body = @{
    symbol        = "BTCUSDT"
    side          = "BUY"
    type          = "LIMIT"
    time_in_force = "GTC"
    price         = 50000
    stop_price    = 0
    quantity      = 1
} | ConvertTo-Json -Compress

Write-Host "Target:   POST $uri"
Write-Host "Requests: $Count (fired concurrently, to actually exceed burst before refill catches up)"
Write-Host "---------------------------------------------------------------"

# Fire all requests near-simultaneously using async HttpClient tasks.
# Sequential requests (even "fast" ones with Invoke-WebRequest) have enough
# gap between them for the token bucket to partially refill, which is why
# a sequential loop can fail to ever trigger a 429 even past the burst size.
Add-Type -AssemblyName System.Net.Http

$client = [System.Net.Http.HttpClient]::new()
$client.DefaultRequestHeaders.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $Token)

$tasks = @()
for ($i = 1; $i -le $Count; $i++) {
    $content = [System.Net.Http.StringContent]::new($body, [System.Text.Encoding]::UTF8, "application/json")
    $tasks += , $client.PostAsync($uri, $content)
}

[System.Threading.Tasks.Task]::WaitAll($tasks)

$allowed = 0
$limited = 0

"{0,-5} {1,-6} {2,-18} {3,-12}" -f "#", "HTTP", "X-RateLimit-Rem", "Retry-After" | Write-Host

for ($i = 0; $i -lt $tasks.Count; $i++) {
    $resp = $tasks[$i].Result
    $status = [int]$resp.StatusCode

    $remaining = "-"
    $retryAfter = "-"
    $remVals = $null
    if ($resp.Headers.TryGetValues("X-RateLimit-Remaining", [ref]$remVals)) { $remaining = $remVals -join "" }
    $retryVals = $null
    if ($resp.Headers.TryGetValues("Retry-After", [ref]$retryVals)) { $retryAfter = $retryVals -join "" }

    "{0,-5} {1,-6} {2,-18} {3,-12}" -f ($i + 1), $status, $remaining, $retryAfter | Write-Host

    if ($status -eq 429) { $limited++ } else { $allowed++ }
}

if ($client) { $client.Dispose() }

Write-Host "---------------------------------------------------------------"
Write-Host "Allowed: $allowed   Rate-limited (429): $limited"

if ($limited -eq 0) {
    Write-Warning "No request was rate-limited across $Count rapid requests."
    Write-Warning "Either raise -Count further, or rate limiting is disabled on this environment."
    exit 2
}