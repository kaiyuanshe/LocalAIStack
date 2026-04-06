. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$studioHome = if ($env:UNSLOTH_STUDIO_HOME) { $env:UNSLOTH_STUDIO_HOME } else { Join-Path (Get-HomeDir) ".unsloth\studio" }

if (-not (Test-Path -LiteralPath $studioHome)) {
  throw "Unsloth Studio home not found: $studioHome"
}

$candidates = @()
$unslothCommand = Get-Command unsloth -ErrorAction SilentlyContinue
if ($unslothCommand) {
  $candidates += $unslothCommand.Source
}
$candidates += @(
  (Join-Path $studioHome "unsloth_studio\Scripts\unsloth.exe"),
  (Join-Path $studioHome "unsloth_studio\Scripts\unsloth.cmd")
)

foreach ($candidate in $candidates) {
  if (Test-Path -LiteralPath $candidate) {
    $pythonBin = $null
    $firstLine = Get-Content -LiteralPath $candidate -TotalCount 1 -ErrorAction SilentlyContinue
    if ($firstLine -and $firstLine.StartsWith("#!")) {
      $pythonBin = $firstLine.Substring(2).Trim()
    }
    if (-not $pythonBin) {
      $pythonBin = Join-Path (Split-Path -Parent $candidate) "python.exe"
    }
    if (-not (Test-Path -LiteralPath $pythonBin)) {
      $pythonBin = Join-Path (Split-Path -Parent $candidate) "python"
    }
    if (Test-Path -LiteralPath $pythonBin) {
      & $pythonBin -c "import importlib.util, sys; raise SystemExit(0 if importlib.util.find_spec('_bz2') is not None else 1)"
      if ($LASTEXITCODE -ne 0) {
        throw "Unsloth Studio Python is missing _bz2: $pythonBin"
      }
    }

    & $candidate studio --help | Out-Null
    if ($LASTEXITCODE -eq 0) {
      Write-Output "Unsloth Studio verification succeeded: $candidate"
      exit 0
    }
  }
}

throw "Unsloth Studio is present under $studioHome, but the CLI could not be launched."
