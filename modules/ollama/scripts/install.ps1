. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Find-OllamaCommand {
  $command = Get-Command ollama -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  $candidates = @(
    (Join-Path $env:LOCALAPPDATA "Programs\Ollama\ollama.exe"),
    (Join-Path $env:LOCALAPPDATA "Programs\Ollama\resources\ollama.exe"),
    (Join-Path (Get-LocalBinDir) "ollama.exe"),
    (Join-Path (Get-HomeDir) "scoop\apps\ollama\current\ollama.exe")
  )
  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path -LiteralPath $candidate)) {
      return $candidate
    }
  }
  return $null
}

function Wait-OllamaApi {
  param([int]$TimeoutSeconds = 30)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    try {
      $response = Invoke-RestMethod -Uri "http://127.0.0.1:11434/api/tags" -Method Get -TimeoutSec 3
      if ($null -ne $response.models) {
        return $true
      }
    } catch {
    }
    Start-Sleep -Seconds 1
  } while ((Get-Date) -lt $deadline)
  return $false
}

function Wait-InstallerProcess {
  param(
    [System.Diagnostics.Process]$Process,
    [int]$TimeoutSeconds = 180
  )

  if ($Process.WaitForExit($TimeoutSeconds * 1000)) {
    return $true
  }
  return $false
}

$existing = Find-OllamaCommand
if ($existing) {
  Write-Output "Ollama already installed: $existing"
  if (-not (Wait-OllamaApi -TimeoutSeconds 5)) {
    Write-Warning "Ollama CLI exists but API is not responding yet. Launch Ollama from the Start menu if needed."
  }
  exit 0
}

$installUrl = if ($env:OLLAMA_INSTALL_URL) { $env:OLLAMA_INSTALL_URL } else { "https://ollama.com/install.ps1" }
$script = Invoke-RestMethod -Uri $installUrl
$tempScript = Join-Path ([System.IO.Path]::GetTempPath()) ("ollama-install-" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
  Set-Content -LiteralPath $tempScript -Value $script -Encoding UTF8
  $shell = Get-Command pwsh.exe -ErrorAction SilentlyContinue
  $process = $null
  if ($shell) {
    $process = Start-Process -FilePath $shell.Source -ArgumentList @("-NoLogo", "-NoProfile", "-NonInteractive", "-File", $tempScript) -PassThru -WindowStyle Hidden
  } else {
    $process = Start-Process -FilePath "powershell.exe" -ArgumentList @("-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", $tempScript) -PassThru -WindowStyle Hidden
  }

  $completed = Wait-InstallerProcess -Process $process -TimeoutSeconds 180
  if (-not $completed) {
    $installed = Find-OllamaCommand
    if ($installed) {
      try {
        $process.Kill()
      } catch {
      }
      Write-Warning "Ollama installer did not exit within the timeout, but Ollama was installed at $installed."
    } else {
      try {
        $process.Kill()
      } catch {
      }
      throw "Ollama installer did not finish within the timeout."
    }
  } elseif ($process.ExitCode -ne 0) {
    throw "Ollama installer exited with code $($process.ExitCode)."
  }
} finally {
  if (Test-Path -LiteralPath $tempScript) {
    Remove-Item -LiteralPath $tempScript -Force
  }
}

$installed = Find-OllamaCommand
if (-not $installed) {
  throw "Ollama install script finished, but ollama was not found in PATH or common install locations."
}

if (-not (Wait-OllamaApi -TimeoutSeconds 30)) {
  Write-Warning "Ollama installed, but the local API at http://127.0.0.1:11434 is not responding yet."
}

Write-Output "Ollama installed at: $installed"
