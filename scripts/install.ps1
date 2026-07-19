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
    # Prefer the latest stable release.
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
                                     -UserAgent 'atl-installer' `
                                     -Headers @{ 'Accept' = 'application/vnd.github+json' }
        $Version = $Release.tag_name
    } catch {
        # /releases/latest 404s when only prereleases exist (a pre-1.0 alpha train);
        # fall through to the full release list below.
        $Version = $null
    }

    # Fallback: no stable release — pick the highest version across ALL releases
    # (prereleases included). The list order is not reliably newest-first, so pick
    # the max by version, never just the first entry. Mirrors install.sh's awk:
    # strip 'v', zero-pad each numeric run to 9 digits, string-compare, take the max.
    if (-not $Version) {
        try {
            $Releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases" `
                                          -UserAgent 'atl-installer' `
                                          -Headers @{ 'Accept' = 'application/vnd.github+json' }
        } catch {
            Write-Error "Could not reach GitHub releases API. Set `$env:ATL_VERSION = 'vX.Y.Z' and re-run. Original error: $_"
        }
        $Version = $Releases |
            Where-Object { $_.tag_name -match '^v[0-9]' } |
            Sort-Object -Property @{ Expression = {
                ([regex]::Matches($_.tag_name.TrimStart('v'), '\d+') |
                    ForEach-Object { '{0:D9}' -f [int]$_.Value }) -join ''
            } } |
            Select-Object -Last 1 -ExpandProperty tag_name
    }

    if (-not $Version) {
        Write-Error 'Could not resolve latest version. Set $env:ATL_VERSION = "vX.Y.Z" and re-run.'
    }
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

# --- verify checksum (fail-closed) ------------------------------------------
# goreleaser publishes atl_<version>_checksums.txt listing "<sha256>  <archive>".
# Verify the downloaded archive before extracting; any failure aborts.
$Checksums     = "atl_${VersionNoV}_checksums.txt"
$ChecksumsUrl  = "https://github.com/$Repo/releases/download/$Version/$Checksums"
$ChecksumsPath = Join-Path $Tmp.FullName $Checksums

Write-Host '→ Verifying checksum'
try {
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing
} catch {
    Write-Error "Could not download checksums file: $_`nURL: $ChecksumsUrl"
}

$Expected = Get-Content $ChecksumsPath |
    Where-Object { $_ -match "\s$([regex]::Escape($Archive))$" } |
    ForEach-Object { ($_ -split '\s+')[0] } |
    Select-Object -First 1
if (-not $Expected) {
    Write-Error "Checksum for $Archive not found in $Checksums."
}

# -ine is load-bearing: Get-FileHash returns UPPERCASE hex, the goreleaser file is
# lowercase, so a case-sensitive compare would false-mismatch on every install.
$Actual = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash
if ($Actual -ine $Expected) {
    Write-Error "Checksum MISMATCH for $Archive — aborting.`n  expected: $Expected`n  actual: $Actual"
}
Write-Host '✓ checksum verified'

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

# Windows locks a running .exe, so a reinstall may need to stop the instance
# holding $Target. Never blind-kill every 'atl': a live `atl work dispatch`
# supervisor owns git worktrees + claude -p workers and must not be torn down.
try {
    Copy-Item -Path $Exe -Destination $Target -Force
} catch {
    # Copy failed — the destination is almost certainly locked by a running atl.
    $running = Get-CimInstance Win32_Process -Filter "Name = 'atl.exe'" -ErrorAction SilentlyContinue

    $dispatch = $running | Where-Object { $_.CommandLine -match '\bwork\s+dispatch\b' }
    if ($dispatch) {
        Write-Error ("Cannot replace $Target while a live 'atl work dispatch' supervisor is running " +
                     "(PID $($dispatch.ProcessId -join ', ')). Stopping it would orphan its in-flight " +
                     "worktrees and workers. Let the sprint finish (or stop it yourself), then re-run.")
    }

    # Only processes actually launched from $Target hold the lock — stop just those,
    # never an unrelated 'atl' (a dev build, another install) elsewhere.
    $holders = $running | Where-Object { $_.ExecutablePath -ieq $Target }
    if ($holders) {
        Write-Host "→ Stopping $($holders.Count) atl instance(s) running from $Target to unlock the file"
        $holders | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
        Start-Sleep -Milliseconds 500
    }

    Copy-Item -Path $Exe -Destination $Target -Force   # retry; if still locked, this throws (honest failure)
}

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
