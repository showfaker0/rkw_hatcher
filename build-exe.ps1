# One-click package: frontend dist embed into backend -> release/rkw_hatcher.exe
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Front = Join-Path $Root "1_front"
$Back = Join-Path $Root "2_back"
$WebDist = Join-Path $Back "web\dist"
$OutDir = Join-Path $Root "release"
$GoBin = "C:\Program Files\Go\bin"

$env:Path = "$GoBin;$env:Path"
$env:VITE_API_BASE = "/api"
$env:GOPROXY = "https://goproxy.cn,direct"

Write-Host "==> 1/3 build frontend (VITE_API_BASE=/api)" -ForegroundColor Cyan
Set-Location $Front
npm run build
if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }

Write-Host "==> 2/3 copy dist -> 2_back/web/dist" -ForegroundColor Cyan
if (Test-Path $WebDist) { Remove-Item -Recurse -Force $WebDist }
New-Item -ItemType Directory -Path $WebDist | Out-Null
Copy-Item -Recurse -Force (Join-Path $Front "dist\*") $WebDist

Write-Host "==> 3/3 build Go exe" -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Set-Location $Back
$exe = Join-Path $OutDir "rkw_hatcher.exe"
go build -ldflags "-s -w" -o $exe ./cmd/server
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Copy-Item -Force (Join-Path $Back ".env.example") (Join-Path $OutDir ".env.example")

Write-Host ""
Write-Host "DONE: $exe" -ForegroundColor Green
Write-Host "Usage:" -ForegroundColor Yellow
Write-Host "  1. Double-click exe; uses/creates rkw_hatcher.db beside it"
Write-Host "  2. Opening another exe force-kills the previous one"
Write-Host "  3. Exit by closing the console window (status light cannot stop packaged backend)"
