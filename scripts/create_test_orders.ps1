<#
.SYNOPSIS
    Creates multiple OPEN LIMIT orders on a running Velocity instance
    and prints their order IDs.

.USAGE
    $env:VELOCITY_TOKEN = "<jwt from identity-service>"

    .\scripts\create_test_orders.ps1

    .\scripts\create_test_orders.ps1 `
        -Count 25 `
        -BaseUrl "http://localhost:8080"

    Optional:
    .\scripts\create_test_orders.ps1 `
        -Count 25 `
        -StartingPrice 200000
#>

param(
    [string]$BaseUrl = $(if ($env:VELOCITY_BASE_URL) {
        $env:VELOCITY_BASE_URL
    } else {
        "http://localhost:8080"
    }),

    [string]$Token = $env:VELOCITY_TOKEN,

    [int]$Count = 25,

    [long]$StartingPrice = 200000
)

if ([string]::IsNullOrWhiteSpace($Token)) {
    Write-Error "Set `$env:VELOCITY_TOKEN to a valid JWT before running this."
    exit 1
}

if ($Count -le 0) {
    Write-Error "-Count must be greater than 0."
    exit 1
}

Add-Type -AssemblyName System.Net.Http

$client = [System.Net.Http.HttpClient]::new()

$client.DefaultRequestHeaders.Authorization =
    [System.Net.Http.Headers.AuthenticationHeaderValue]::new(
        "Bearer",
        $Token
    )

$uri = "$BaseUrl/api/orders"

Write-Host ""
Write-Host "Velocity Test Order Generator"
Write-Host "---------------------------------------------------------------"
Write-Host "Target:        POST $uri"
Write-Host "Orders:        $Count"
Write-Host "Starting price: $StartingPrice"
Write-Host "---------------------------------------------------------------"
Write-Host ""

$successfulIds = @()
$failed = 0

for ($i = 0; $i -lt $Count; $i++) {

    # Unique price for every order.
    $price = $StartingPrice + $i

    # IMPORTANT:
    # Velocity expects numeric int64 values for price, stop_price
    # and quantity. Do not send these as strings.
    $bodyObject = @{
        symbol        = "BTCUSDT"
        side          = "BUY"
        type          = "LIMIT"
        time_in_force = "GTC"
        price         = [long]$price
        stop_price    = [long]0
        quantity      = [long]1
    }

    $body = $bodyObject | ConvertTo-Json -Compress

    $content = $null

    try {

        $content = [System.Net.Http.StringContent]::new(
            $body,
            [System.Text.Encoding]::UTF8,
            "application/json"
        )

        $response = $client.PostAsync(
            $uri,
            $content
        ).GetAwaiter().GetResult()

        $status = [int]$response.StatusCode

        $responseBody = $response.Content.ReadAsStringAsync().
            GetAwaiter().GetResult()

        if ($status -ge 200 -and $status -lt 300) {

            try {
                $json = $responseBody | ConvertFrom-Json

                $orderId = $json.data.order_id

                if ($null -ne $orderId) {

                    $successfulIds += [long]$orderId

                    Write-Host (
                        "{0,-5} {1,-6} price={2,-10} order_id={3}" -f `
                        ($i + 1),
                        $status,
                        $price,
                        $orderId
                    )
                }
                else {
                    Write-Warning (
                        "Request $($i + 1) succeeded but no order_id was returned."
                    )
                    $failed++
                }
            }
            catch {
                Write-Warning (
                    "Request $($i + 1) returned invalid JSON: $responseBody"
                )
                $failed++
            }
        }
        else {

            Write-Host (
                "{0,-5} {1,-6} price={2,-10} FAILED" -f `
                ($i + 1),
                $status,
                $price
            )

            Write-Host "      Response: $responseBody"

            $failed++
        }
    }
    catch {

        Write-Host (
            "{0,-5} ERROR  price={1,-10}" -f `
            ($i + 1),
            $price
        )

        Write-Host "      $($_.Exception.Message)"

        $failed++
    }
    finally {

        # Prevent null-valued Dispose exception.
        if ($null -ne $content) {
            $content.Dispose()
        }
    }
}

$client.Dispose()

Write-Host ""
Write-Host "==============================================================="
Write-Host "RESULT"
Write-Host "==============================================================="
Write-Host "Requested:  $Count"
Write-Host "Created:    $($successfulIds.Count)"
Write-Host "Failed:     $failed"
Write-Host "==============================================================="

if ($successfulIds.Count -gt 0) {

    Write-Host ""
    Write-Host "Order IDs:"
    Write-Host ""

    foreach ($id in $successfulIds) {
        Write-Host $id
    }

    Write-Host ""
    Write-Host "---------------------------------------------------------------"
    Write-Host "COPY/PASTE FOR DELETE RATE-LIMIT TEST:"
    Write-Host "---------------------------------------------------------------"
    Write-Host ""

    $formattedIds = $successfulIds -join ",`n        "

    Write-Host ".\scripts\loadtest_cancel_rate_limit.ps1 ``"
    Write-Host '    -BaseUrl "http://localhost:8080" ``'
    Write-Host "    -OrderIds @("
    Write-Host "        $formattedIds"
    Write-Host "    )"

    Write-Host ""
}

if ($successfulIds.Count -eq 0) {
    Write-Error "No test orders were created."
    exit 2
}

if ($successfulIds.Count -lt $Count) {
    Write-Warning "Only $($successfulIds.Count) of $Count requested orders were created."
}