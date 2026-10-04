Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  Starting GOWA Unified Gateway (Backend + Glass UI)   " -ForegroundColor Green
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  Dashboard is live at: http://localhost:3000          " -ForegroundColor Yellow
Write-Host "========================================================" -ForegroundColor Cyan

Set-Location (Join-Path $PSScriptRoot "src")
go run -tags purego . rest
