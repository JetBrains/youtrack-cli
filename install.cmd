@echo off
:: Copyright 2021-2026 JetBrains s.r.o.
::
:: Licensed under the Apache License, Version 2.0 (the "License");
:: you may not use this file except in compliance with the License.
:: You may obtain a copy of the License at
::
:: https://www.apache.org/licenses/LICENSE-2.0
::
:: Unless required by applicable law or agreed to in writing, software
:: distributed under the License is distributed on an "AS IS" BASIS,
:: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
:: See the License for the specific language governing permissions and
:: limitations under the License.
::
:: Use the PowerShell installer for download verification, staging, cleanup,
:: architecture detection, and PATH configuration.
setlocal
set "YOUTRACK_INSTALL_RELEASE=%~1"
set "YOUTRACK_INSTALL_DIR=%~2"
set "YOUTRACK_INSTALL_PS1=%~dp0install.ps1"
set "YOUTRACK_INSTALL_BOOTSTRAP=%TEMP%\youtrack_install_%RANDOM%%RANDOM%.ps1"

if exist "%YOUTRACK_INSTALL_PS1%" goto :install

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
    "$ErrorActionPreference = 'Stop';" ^
    "try {" ^
    "  if (Get-Command gh -ErrorAction SilentlyContinue) {" ^
    "    try {" ^
    "      & gh release download --repo JetBrains/youtrack-cli --pattern install.ps1 --output $env:YOUTRACK_INSTALL_BOOTSTRAP --clobber;" ^
    "      if ($LASTEXITCODE -eq 0) { exit 0 };" ^
    "    } catch { Write-Verbose 'gh download failed; trying GitHub HTTPS.' };" ^
    "  };" ^
    "  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12;" ^
    "  $url = 'https://github.com/JetBrains/youtrack-cli/releases/latest/download/install.ps1';" ^
    "  Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $env:YOUTRACK_INSTALL_BOOTSTRAP;" ^
    "} catch { Write-Error $_; exit 1 }"
if errorlevel 1 (
    del "%YOUTRACK_INSTALL_BOOTSTRAP%" 2>nul
    exit /b 1
)
set "YOUTRACK_INSTALL_PS1=%YOUTRACK_INSTALL_BOOTSTRAP%"

:install
powershell -NoProfile -ExecutionPolicy Bypass -File "%YOUTRACK_INSTALL_PS1%"
set "INSTALL_STATUS=%ERRORLEVEL%"
if exist "%YOUTRACK_INSTALL_BOOTSTRAP%" del "%YOUTRACK_INSTALL_BOOTSTRAP%"
exit /b %INSTALL_STATUS%
