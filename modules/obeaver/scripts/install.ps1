. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Get-EBeaverHome {
  if ($env:OBEAVER_HOME) {
    return $env:OBEAVER_HOME
  }
  return Join-Path (Get-LocalAIStackDir) "tools\obeaver"
}

function Get-EBeaverRepoDir {
  return Join-Path (Get-EBeaverHome) "repo"
}

function Get-EBeaverVenvDir {
  return Join-Path (Get-EBeaverHome) ".venv"
}

function Get-EBeaverPython {
  return Join-Path (Get-EBeaverVenvDir) "Scripts\python.exe"
}

function Get-EBeaverWrapperPath {
  return Join-Path (Get-LocalBinDir) "obeaver.cmd"
}

function Get-EBeaverCliPath {
  $candidates = @(
    (Join-Path (Get-EBeaverVenvDir) "Scripts\obeaver.exe"),
    (Join-Path (Get-EBeaverVenvDir) "Scripts\obeaver.cmd"),
    (Join-Path (Get-EBeaverVenvDir) "Scripts\obeaver")
  )

  foreach ($candidate in $candidates) {
    if (Test-Path -LiteralPath $candidate) {
      return $candidate
    }
  }

  throw "oBeaver CLI entrypoint was not found in the virtual environment."
}

function Ensure-Git {
  if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    throw "git is required but was not found in PATH."
  }
}

function Ensure-FoundryLocal {
  $foundry = Get-Command foundry -ErrorAction SilentlyContinue
  if ($foundry) {
    Write-Output "Foundry Local already available: $($foundry.Source)"
    return
  }

  $winget = Get-Command winget -ErrorAction SilentlyContinue
  if (-not $winget) {
    throw "Foundry Local is required on Windows for the default oBeaver engine, but winget was not found. Install Foundry Local manually with 'winget install Microsoft.FoundryLocal' and rerun the install."
  }

  Write-Output "Installing Foundry Local via winget..."
  & $winget.Source install --id Microsoft.FoundryLocal --exact --accept-source-agreements --accept-package-agreements
  if ($LASTEXITCODE -ne 0) {
    throw "winget failed to install Microsoft.FoundryLocal (exit code $LASTEXITCODE)."
  }

  $foundry = Get-Command foundry -ErrorAction SilentlyContinue
  if (-not $foundry) {
    throw "Foundry Local install finished, but 'foundry' is still not available in PATH. Open a new shell or add it to PATH, then rerun the install."
  }
}

function Write-EBeaverWrapper {
  $wrapperPath = Get-EBeaverWrapperPath
  $cliPath = Get-EBeaverCliPath
  Ensure-Directory -Path (Split-Path -Parent $wrapperPath)
  $content = @"
@echo off
"$cliPath" %*
"@
  Set-Content -LiteralPath $wrapperPath -Value $content -Encoding ASCII
}

$repoUrl = if ($env:OBEAVER_REPO_URL) { $env:OBEAVER_REPO_URL } else { "https://github.com/microsoft/obeaver.git" }
$repoRef = if ($env:OBEAVER_REPO_REF) { $env:OBEAVER_REPO_REF } else { "main" }
$homeDir = Get-EBeaverHome
$repoDir = Get-EBeaverRepoDir
$venvDir = Get-EBeaverVenvDir
$venvPython = Get-EBeaverPython

Ensure-Git
Ensure-FoundryLocal
Ensure-Directory -Path $homeDir

if (-not (Test-Path -LiteralPath (Join-Path $repoDir ".git"))) {
  & git clone --depth 1 --branch $repoRef $repoUrl $repoDir
  if ($LASTEXITCODE -ne 0) {
    throw "git clone failed with code $LASTEXITCODE."
  }
} else {
  Write-Output "oBeaver repository already present: $repoDir"
}

if (-not (Test-Path -LiteralPath $venvPython)) {
  Invoke-Python -Arguments @("-m", "venv", $venvDir)
}

& $venvPython -m pip install --upgrade pip setuptools wheel
if ($LASTEXITCODE -ne 0) {
  throw "Failed to upgrade pip/setuptools/wheel in the oBeaver virtual environment."
}

& $venvPython -m pip install -e $repoDir
if ($LASTEXITCODE -ne 0) {
  throw "Failed to install oBeaver in editable mode."
}

Write-EBeaverWrapper

& (Get-EBeaverCliPath) version | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw "oBeaver installed, but the CLI failed to start."
}

Write-Output "oBeaver installed at: $(Get-EBeaverWrapperPath)"
