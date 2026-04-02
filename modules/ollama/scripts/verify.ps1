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

try {
  $response = Invoke-RestMethod -Uri "http://127.0.0.1:11434/api/tags" -Method Get -TimeoutSec 5
} catch {
  throw "Failed to reach Ollama API at http://127.0.0.1:11434/api/tags."
}

if ($null -eq $response.models) {
  throw "Ollama API response did not include models."
}

if (-not (Find-OllamaCommand)) {
  throw "Ollama CLI is not available in PATH or common install locations."
}

$ollama = Find-OllamaCommand
& $ollama --version | Out-Null
Write-Output "Ollama verification succeeded."
