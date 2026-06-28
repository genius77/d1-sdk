<#
.SYNOPSIS
    D1 动态库下载脚本 (Windows PowerShell)
.DESCRIPTION
    从 GitHub Release 下载对应平台的 D1 动态库
.PARAMETER Version
    目标版本号，不指定则下最新版
.EXAMPLE
    .\download_d1.ps1
    .\download_d1.ps1 -Version v1.1.0
#>
param([string]$Version = "latest")

$ErrorActionPreference = "Stop"
$D1Repo = "genius77/D1"
$DepsDir = "deps"
$Platform = "windows-x64"

Write-Host ""; Write-Host "D1 动态库下载工具" -ForegroundColor Green; Write-Host ""

# ─── 版本解析 ──────────────────────────────────────────────
if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$D1Repo/releases/latest"
    $Version = $release.tag_name
}
Write-Host "✓ 版本: $Version" -ForegroundColor Green

# ─── 下载 ──────────────────────────────────────────────────
$assetName = "libd1-$Platform-$Version"
$data = Invoke-RestMethod -Uri "https://api.github.com/repos/$D1Repo/releases/tags/$Version"
$asset = $data.assets | Where-Object { $_.name -like "$assetName*" } | Select-Object -First 1

if (-not $asset) { Write-Host "✗ 未找到产物" -ForegroundColor Red; exit 1 }

New-Item -ItemType Directory -Force -Path $DepsDir | Out-Null
$zip = Join-Path $DepsDir "$assetName.zip"
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $DepsDir -Force
Remove-Item $zip

Write-Host ""
Write-Host "✅ D1 下载完成，文件位于 .\$DepsDir\" -ForegroundColor Green
Write-Host "  → 接下来: cd examples\csharp\01_hello_d1 && dotnet run"