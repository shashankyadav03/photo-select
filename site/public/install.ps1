# Photo Select installer for Windows.
# Usage (PowerShell): $u='https://<site>'; irm $u/install.ps1 | iex
# Installs into %LOCALAPPDATA%\PhotoSelect, adds a Desktop shortcut, then starts the app.
# Files fetched this way carry no "downloaded from the internet" mark, so SmartScreen doesn't block it.
& {
  $ErrorActionPreference = 'Stop'
  $ProgressPreference = 'SilentlyContinue'  # makes Invoke-WebRequest much faster on Windows PowerShell 5

  $base = if ($u) { $u } else { $env:PHOTO_SELECT_URL }
  if (-not $base) {
    Write-Host 'Copy the full command from the Photo Select website and paste it again.' -ForegroundColor Red
    return
  }
  $base = $base.TrimEnd('/')
  $dest = Join-Path $env:LOCALAPPDATA 'PhotoSelect'
  $tmp = Join-Path ([IO.Path]::GetTempPath()) ('photo-select-' + [guid]::NewGuid())
  New-Item -ItemType Directory -Force -Path $tmp, $dest | Out-Null

  try {
    Write-Host ''
    Write-Host '  Downloading Photo Select for Windows...'
    Invoke-WebRequest -UseBasicParsing -Uri "$base/downloads/photo-select-windows.zip" -OutFile "$tmp\app.zip"
    Expand-Archive -Path "$tmp\app.zip" -DestinationPath "$tmp\x" -Force
    $exe = Get-ChildItem -Path "$tmp\x" -Recurse -Filter 'photo-select.exe' | Select-Object -First 1
    if (-not $exe) { throw 'The download looks broken. Please try again.' }

    # An older copy that is still running would lock the file. Progress is saved after every swipe.
    Get-Process -Name 'photo-select' -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Milliseconds 300
    Copy-Item -Path $exe.FullName -Destination "$dest\photo-select.exe" -Force
  } finally {
    Remove-Item -Recurse -Force -Path $tmp -ErrorAction SilentlyContinue
  }

  $desktop = [Environment]::GetFolderPath('Desktop')
  $shell = New-Object -ComObject WScript.Shell
  $lnk = $shell.CreateShortcut((Join-Path $desktop 'Photo Select.lnk'))
  $lnk.TargetPath = "$dest\photo-select.exe"
  $lnk.WorkingDirectory = $dest
  $lnk.Description = 'Swipe through your photos and keep the best ones'
  $lnk.Save()

  Write-Host ''
  Write-Host "  Installed in $dest" -ForegroundColor Green
  Write-Host '  Next time: double-click "Photo Select" on your Desktop'
  Write-Host ''
  Write-Host '  Starting it now. A black window opens - keep it open while you swipe.'
  Write-Host '  If Windows Firewall asks, tick "Private networks" and click Allow access.'
  Write-Host ''
  Start-Process -FilePath "$dest\photo-select.exe" -WorkingDirectory $dest
}
