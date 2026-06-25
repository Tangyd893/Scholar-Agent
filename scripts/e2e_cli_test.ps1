# ScholarAgent - CLI E2E Test (Phase 1)
# Usage: powershell -ExecutionPolicy Bypass -File scripts/e2e_cli_test.ps1

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " ScholarAgent CLI E2E Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$sessionFile = ".e2e_session.txt"
$pass = 0
$fail = 0

# Test 1
Write-Host "[1/3] First query..." -ForegroundColor Yellow
$output1 = & go run ./agent-core/cmd/cli --query "find attention mechanism papers" --mock *>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAIL: crashed (exit $LASTEXITCODE)" -ForegroundColor Red
    $fail++
} elseif ($output1 -match "answer") {
    if ($output1 -match "sess_(\d+)") {
        $sessionID = $matches[1]
        "sess_$sessionID" | Out-File $sessionFile
        Write-Host "  Session: sess_$sessionID" -ForegroundColor Green
    }
    Write-Host "  PASS" -ForegroundColor Green
    $pass++
} else {
    Write-Host "  FAIL: no answer event" -ForegroundColor Red
    $fail++
}

# Test 2
Write-Host "[2/3] Follow-up..." -ForegroundColor Yellow
$sessionID = Get-Content $sessionFile
$output2 = & go run ./agent-core/cmd/cli --query "which one is most classic?" --mock --session "$sessionID" *>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAIL: crashed" -ForegroundColor Red
    $fail++
} elseif ($output2 -match "answer") {
    Write-Host "  PASS" -ForegroundColor Green
    $pass++
} else {
    Write-Host "  FAIL: no answer" -ForegroundColor Red
    $fail++
}

# Test 3
Write-Host "[3/3] Third query..." -ForegroundColor Yellow
$output3 = & go run ./agent-core/cmd/cli --query "recommend more papers" --mock --session "$sessionID" *>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAIL: crashed" -ForegroundColor Red
    $fail++
} elseif ($output3 -match "answer") {
    Write-Host "  PASS" -ForegroundColor Green
    $pass++
} else {
    Write-Host "  FAIL: no answer" -ForegroundColor Red
    $fail++
}

Remove-Item $sessionFile -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Result: $pass PASS, $fail FAIL" -ForegroundColor $(if ($fail -eq 0) { "Green" } else { "Red" })
Write-Host "========================================" -ForegroundColor Cyan

exit $fail
