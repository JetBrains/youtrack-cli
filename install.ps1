#
# Copyright 2021-2026 JetBrains s.r.o.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$repo = "JetBrains/youtrack-cli"
$binName = "youtrack.exe"
$Release = if ($env:YOUTRACK_INSTALL_RELEASE) { $env:YOUTRACK_INSTALL_RELEASE } else { "" }
$OutDir = if ($env:YOUTRACK_INSTALL_DIR) { $env:YOUTRACK_INSTALL_DIR } else { "$HOME\.local\bin" }
$signingKeyFingerprint = "B46DC71E03FEEB7F89D1F2491F7A8F87B9D8F501"
$tempDir = $null
$stagedBin = $null

function Cleanup {
    if ($tempDir -and (Test-Path -LiteralPath $tempDir)) {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($stagedBin -and (Test-Path -LiteralPath $stagedBin)) {
        Remove-Item -LiteralPath $stagedBin -Force -ErrorAction SilentlyContinue
    }
}

function Resolve-LatestRelease {
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        try {
            $tag = & gh api "repos/$repo/releases/latest" --jq '.tag_name'
            if ($LASTEXITCODE -eq 0 -and $tag) { return $tag.Trim() }
        } catch { Write-Verbose "gh could not resolve the release; trying GitHub HTTPS." }
    }
    return (Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest").tag_name
}

function Download-ReleaseAsset {
    param([string]$Name, [string]$Destination)
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        try {
            & gh release download $Release --repo $repo --pattern $Name --output $Destination --clobber
            if ($LASTEXITCODE -eq 0) { return }
        } catch { Write-Verbose "gh could not download $Name; trying GitHub HTTPS." }
    }
    Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/$repo/releases/download/$Release/$Name" -OutFile $Destination
}

function Verify-ChecksumSignature {
    param([string]$Checksums)
    if (-not (Get-Command gpg -ErrorAction SilentlyContinue)) {
        Write-Warning "gpg not found, checksums.txt signature not checked."
        return
    }
    if ((Get-Item -LiteralPath "$Checksums.asc").Length -eq 0) {
        throw "checksums.txt.asc is empty: this release is not signed"
    }
    $gpgHome = Join-Path $tempDir "gnupg"
    New-Item -ItemType Directory -Path $gpgHome | Out-Null
    $keyFile = Join-Path $tempDir "signing-key.asc"
    # Same embedded public key as install.sh; VALIDSIG below pins its fingerprint.
    @'
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=IaXJ
-----END PGP PUBLIC KEY BLOCK-----
'@ | Set-Content -LiteralPath $keyFile -Encoding Ascii
    & gpg --homedir $gpgHome --batch --quiet --logger-fd 1 --import $keyFile | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to import the release signing key $signingKeyFingerprint" }
    $status = & gpg --homedir $gpgHome --batch --logger-fd 1 --status-fd 1 --verify "$Checksums.asc" $Checksums
    $verified = $false
    foreach ($line in $status) {
        $fields = $line -split '\s+'
        if ($fields.Length -ge 3 -and $fields[0] -eq '[GNUPG:]' -and $fields[1] -eq 'VALIDSIG' -and
            ($fields[2] -eq $signingKeyFingerprint -or $fields[-1] -eq $signingKeyFingerprint)) {
            $verified = $true
        }
    }
    if ($LASTEXITCODE -ne 0 -or -not $verified) {
        throw "signature verification failed for checksums.txt (expected key $signingKeyFingerprint)"
    }
}

function Verify-Checksum {
    param([string]$Archive, [string]$Name, [string]$Checksums)
    $expected = $null
    $pattern = '^([0-9a-fA-F]{64})\s+\*?' + [regex]::Escape($Name) + '$'
    foreach ($line in Get-Content -LiteralPath $Checksums) {
        if ($line -match $pattern) { $expected = $Matches[1]; break }
    }
    if (-not $expected) { throw "checksum entry not found for $Name" }
    if ((Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash -ne $expected) {
        throw "checksum verification failed for $Name"
    }
}

# 'youtrack update' sets YOUTRACK_INSTALL_UPDATE: the user already knows the installer and the next steps.
$isUpdate = [bool]$env:YOUTRACK_INSTALL_UPDATE
if (-not $isUpdate) {
    Write-Host "`nYouTrack CLI (installer)`n"
    Write-Host "This script installs YouTrack CLI to $OutDir\$binName`n"
    Write-Host "To install a specific version, set YOUTRACK_INSTALL_RELEASE; to choose a directory, set YOUTRACK_INSTALL_DIR.`n"
}

try {
    if (-not $Release) { $Release = Resolve-LatestRelease }
    if ($Release -notmatch '^v?[0-9A-Za-z._-]+$') { throw "invalid release: $Release" }
    $version = $Release -replace '^v', ''
    $nativeArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $arch = switch ($nativeArch) {
        'ARM64' { 'arm64' }
        'AMD64' { 'x86_64' }
        default { throw "unsupported architecture: $nativeArch" }
    }
    $assetName = "youtrack_$($version)_windows_$($arch).zip"
    Write-Host "Installing youtrack ($Release)`n"

    if (-not (Test-Path -LiteralPath $OutDir)) {
        New-Item -ItemType Directory -Path $OutDir | Out-Null
    }
    if (-not (Test-Path -LiteralPath $OutDir -PathType Container)) {
        throw "output path is not a directory: $OutDir"
    }
    $OutDir = (Resolve-Path -LiteralPath $OutDir).ProviderPath
    $tempDir = Join-Path ([IO.Path]::GetTempPath()) "youtrack_$([guid]::NewGuid().ToString('N'))"
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    $tempZip = Join-Path $tempDir $assetName
    $checksums = Join-Path $tempDir "checksums.txt"
    Download-ReleaseAsset $assetName $tempZip
    Download-ReleaseAsset 'checksums.txt' $checksums
    Download-ReleaseAsset 'checksums.txt.asc' "$checksums.asc"
    Verify-ChecksumSignature $checksums
    Verify-Checksum $tempZip $assetName $checksums

    $tempExtract = Join-Path $tempDir "extract"
    Expand-Archive -LiteralPath $tempZip -DestinationPath $tempExtract -Force
    $exePath = Get-ChildItem -LiteralPath $tempExtract -Filter $binName -Recurse -File | Select-Object -First 1
    if (-not $exePath) { throw "could not find $binName in the downloaded archive" }
    $stagedBin = Join-Path $OutDir ".youtrack_staged_$([guid]::NewGuid().ToString('N')).exe"
    Copy-Item -LiteralPath $exePath.FullName -Destination $stagedBin
    $target = Join-Path $OutDir $binName
    # Earlier updates may have left old binaries behind; remove the ones no longer running.
    Get-ChildItem -LiteralPath $OutDir -Filter "$binName.old*" -File -ErrorAction SilentlyContinue |
        Remove-Item -Force -ErrorAction SilentlyContinue
    $oldBin = "$target.old"
    # A youtrack.exe.old still locked by a running process can't be replaced, so set this binary aside under a fresh name.
    if (Test-Path -LiteralPath $oldBin) { $oldBin = "$target.old-$([guid]::NewGuid().ToString('N'))" }
    # Windows locks a running executable against overwrite but allows a rename, so move the
    # current binary aside first. This lets 'youtrack update' replace the running youtrack.exe.
    if (Test-Path -LiteralPath $target) { Move-Item -LiteralPath $target -Destination $oldBin -Force }
    try {
        Move-Item -LiteralPath $stagedBin -Destination $target -Force
    } catch {
        if (Test-Path -LiteralPath $oldBin) { Move-Item -LiteralPath $oldBin -Destination $target -Force }
        throw
    }
    $stagedBin = $null
    # Fails while the old binary still runs; the next install removes it.
    Remove-Item -LiteralPath $oldBin -Force -ErrorAction SilentlyContinue

    Write-Host "`nInstalled at $OutDir\$binName`n"
    & (Join-Path $OutDir $binName) --version
    if ($LASTEXITCODE -ne 0) { throw "installed binary failed to run" }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (@($userPath -split ';' | ForEach-Object { $_.TrimEnd('\') }) -notcontains $OutDir.TrimEnd('\')) {
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$OutDir", 'User')
        Write-Host "Added $OutDir to your user PATH. Restart your terminal for changes to take effect."
    }
    if (@($env:Path -split ';' | ForEach-Object { $_.TrimEnd('\') }) -notcontains $OutDir.TrimEnd('\')) {
        $env:Path += ";$OutDir"
    }
    if (-not $isUpdate) {
        Write-Host "`nNext steps:"
        Write-Host "  youtrack instance add"
        Write-Host "  youtrack user current"
        Write-Host "  youtrack --help`n"
    }
} catch {
    Write-Error "Error: $($_.Exception.Message)" -ErrorAction Continue
    exit 1
} finally {
    Cleanup
}
