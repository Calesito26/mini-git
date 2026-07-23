$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$installDir = Join-Path $env:USERPROFILE "go\bin"
$outputExe = Join-Path $installDir "minigit.exe"

Set-Location $projectRoot
New-Item -ItemType Directory -Path $installDir -Force | Out-Null

go build -o $outputExe .\cmd\minigit

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $installDir) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Se agrego $installDir al PATH del usuario."
    Write-Host "Abre una terminal nueva para usar minigit desde cualquier carpeta."
}

Write-Host "Mini-Git instalado en $outputExe"
Write-Host "Prueba con: minigit help"
