function spf() {
    param (
        [string[]]$Params
    )
    $spf_location = [Environment]::GetFolderPath("LocalApplicationData") + "\Programs\orbit\orbit.exe"
    $SPF_LAST_DIR_PATH = [Environment]::GetFolderPath("LocalApplicationData") + "\orbit\lastdir"

    & $spf_location @Params

    if (Test-Path $SPF_LAST_DIR_PATH) {
        $SPF_LAST_DIR = Get-Content -Path $SPF_LAST_DIR_PATH
        Invoke-Expression $SPF_LAST_DIR
        Remove-Item -Force $SPF_LAST_DIR_PATH
    }
}
