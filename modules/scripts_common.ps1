Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-HomeDir {
  return [Environment]::GetFolderPath("UserProfile")
}

function Get-LocalAIStackDir {
  return Join-Path (Get-HomeDir) ".localaistack"
}

function Get-LocalShareDir {
  return Join-Path (Get-HomeDir) ".local\share"
}

function Get-LocalBinDir {
  return Join-Path (Get-HomeDir) ".local\bin"
}

function Ensure-Directory {
  param([string]$Path)
  New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

function Get-PythonInvocation {
  if (Get-Command py -ErrorAction SilentlyContinue) {
    return @("py", "-3")
  }
  if (Get-Command python -ErrorAction SilentlyContinue) {
    return @("python")
  }
  if (Get-Command python3 -ErrorAction SilentlyContinue) {
    return @("python3")
  }
  throw "Python 3 is required but was not found in PATH."
}

function Invoke-Python {
  param([string[]]$Arguments)
  $python = Get-PythonInvocation
  $pythonExe = $python[0]
  $pythonArgs = @()
  if ($python.Count -gt 1) {
    $pythonArgs = $python[1..($python.Count - 1)]
  }
  & $pythonExe @pythonArgs @Arguments
}

function Invoke-PythonCode {
  param([string]$Code)
  Invoke-Python -Arguments @("-c", $Code)
}

function Remove-IfExists {
  param([string]$Path)
  if (Test-Path -LiteralPath $Path) {
    Remove-Item -LiteralPath $Path -Recurse -Force
  }
}

function Write-UnsupportedWindowsScript {
  param(
    [string]$ModuleName,
    [string]$ScriptName,
    [string]$Reason
  )
  if ([string]::IsNullOrWhiteSpace($Reason)) {
    $Reason = "This module does not yet provide a Windows implementation."
  }
  throw "$ModuleName/$ScriptName is not supported on Windows. $Reason"
}

function Get-ComfyUIHome {
  if ($env:COMFYUI_HOME) {
    return $env:COMFYUI_HOME
  }
  return Join-Path (Get-LocalShareDir) "localaistack\comfyui"
}

function Get-ModelsHome {
  if ($env:MODELS_HOME) {
    return $env:MODELS_HOME
  }
  return Join-Path (Get-LocalAIStackDir) "models"
}

function New-FileLinkOrCopy {
  param(
    [string]$Target,
    [string]$LinkPath
  )

  if (Test-Path -LiteralPath $LinkPath) {
    Remove-Item -LiteralPath $LinkPath -Force
  }

  $parent = Split-Path -Parent $LinkPath
  Ensure-Directory -Path $parent

  try {
    New-Item -ItemType SymbolicLink -Path $LinkPath -Target $Target -Force | Out-Null
    return
  } catch {
  }

  try {
    New-Item -ItemType HardLink -Path $LinkPath -Target $Target -Force | Out-Null
    return
  } catch {
  }

  Copy-Item -LiteralPath $Target -Destination $LinkPath -Force
}

function Get-PipUserSiteScriptsPath {
  if (-not (Get-Command py -ErrorAction SilentlyContinue) -and -not (Get-Command python -ErrorAction SilentlyContinue) -and -not (Get-Command python3 -ErrorAction SilentlyContinue)) {
    return $null
  }
  $code = "import site; print(site.USER_BASE)"
  $output = Invoke-Python -Arguments @("-c", $code) | Select-Object -Last 1
  if ($null -eq $output) {
    return $null
  }
  $userBase = $output.Trim()
  if ([string]::IsNullOrWhiteSpace($userBase)) {
    return $null
  }
  return Join-Path $userBase "Scripts"
}
