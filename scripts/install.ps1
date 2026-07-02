# One-liner installer for atl (Windows, PowerShell).
#
# Usage:
#   irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
#
# Env overrides (set before piping):
#   $env:ATL_VERSION = 'v2.0.0'                  # pin a specific release
#   $env:ATL_INSTALL_DIR = 'C:\Users\<you>\bin'  # where to drop atl.exe
#
# Installs atl.exe into $env:ATL_INSTALL_DIR (default:
# %LOCALAPPDATA%\Programs\atl) and adds that directory to the user PATH if it
# isn't already there. No admin rights, no package-manager prerequisites.

$ErrorActionPreference = 'Stop'

$Repo        = 'agentteamland/atl'
$BinaryName  = 'atl.exe'
$DefaultDir  = Join-Path $env:LOCALAPPDATA 'Programs\atl'
$InstallDir  = if ($env:ATL_INSTALL_DIR) { $env:ATL_INSTALL_DIR } else { $DefaultDir }
$Version     = $env:ATL_VERSION

# --- arch detection ---------------------------------------------------------

$Arch = switch -Regex ($env:PROCESSOR_ARCHITECTURE) {
    '(AMD64|x86_64)' { 'amd64' }
    'ARM64'          { 'arm64' }
    default          {
        Write-Error "Unsupported processor architecture: $env:PROCESSOR_ARCHITECTURE (supported: AMD64, ARM64)"
    }
}

# --- resolve latest version -------------------------------------------------

if (-not $Version) {
    Write-Host '→ Resolving latest release...'
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
                                     -UserAgent 'atl-installer' `
                                     -Headers @{ 'Accept' = 'application/vnd.github+json' }
        $Version = $Release.tag_name
    } catch {
        Write-Error "Could not reach GitHub releases API. Set `$env:ATL_VERSION = 'vX.Y.Z' and re-run. Original error: $_"
    }
}

if (-not $Version) {
    Write-Error 'Version resolution returned empty. Set $env:ATL_VERSION = "vX.Y.Z" and re-run.'
}

$VersionNoV = $Version.TrimStart('v')
$Archive    = "atl_${VersionNoV}_windows_${Arch}.zip"
$Url        = "https://github.com/$Repo/releases/download/$Version/$Archive"

# --- download + extract -----------------------------------------------------

Write-Host "→ Downloading $Url"

$Tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.IO.Path]::GetRandomFileName())) -Force
$ArchivePath = Join-Path $Tmp.FullName $Archive

try {
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing
} catch {
    Write-Error "Download failed: $_`nURL: $Url"
}

Write-Host "→ Extracting to $Tmp"
Expand-Archive -Path $ArchivePath -DestinationPath $Tmp.FullName -Force

$Exe = Join-Path $Tmp.FullName $BinaryName
if (-not (Test-Path $Exe)) {
    Write-Error "Extracted archive did not contain $BinaryName."
}

# --- install + PATH ---------------------------------------------------------

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$Target = Join-Path $InstallDir $BinaryName
Write-Host "→ Installing to $Target"

# Stop any running atl.exe so the overwrite can succeed.
Get-Process -Name 'atl' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

Copy-Item -Path $Exe -Destination $Target -Force

# Ensure install dir is on the user PATH.
$UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not $UserPath) { $UserPath = '' }

$PathEntries = $UserPath -split ';' | Where-Object { $_ -ne '' }
if ($PathEntries -notcontains $InstallDir) {
    Write-Host "→ Adding $InstallDir to user PATH"
    $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable('Path', $NewPath, 'User')
    $env:Path = "$env:Path;$InstallDir"
    $PathMessage = "PATH updated (user scope). Open a new terminal for it to apply everywhere."
} else {
    $PathMessage = "$InstallDir already on PATH."
}

# --- verify + cleanup -------------------------------------------------------

Write-Host ''
Write-Host "✓ atl $Version installed to $Target" -ForegroundColor Green
Write-Host $PathMessage
Write-Host ''

try {
    & $Target --version
} catch {
    Write-Warning "Installed, but could not run atl --version: $_"
}

Remove-Item -Recurse -Force $Tmp.FullName -ErrorAction SilentlyContinue

Write-Host ''
Write-Host 'Next: cd into a project and run:'
Write-Host '  atl search            # browse the team catalog'
Write-Host '  atl install <handle>/<team>'
