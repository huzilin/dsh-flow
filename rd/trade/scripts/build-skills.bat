@echo off
setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
set "REGISTRY_DIR=%PROJECT_ROOT%\skills\registry"
set "PLUGIN_DIR=%PROJECT_ROOT%\skills\plugin"

mkdir "%PLUGIN_DIR%" 2>nul

if not exist "%REGISTRY_DIR%" (
    echo Error: Registry directory not found at %REGISTRY_DIR%
    exit /b 1
)

for %%f in ("%REGISTRY_DIR%\*.yaml" "%REGISTRY_DIR%\*.yml") do (
    echo Processing %%~nxf...
    
    set "name="
    set "repo_url="
    set "local_path="
    set "branch="
    set "version="
    
    for /f "tokens=1,* delims=:" %%a in ('findstr /i "^name:" "%%f"') do (
        set "name=%%b"
        set "name=!name:~1!"
    )
    for /f "tokens=1,* delims=:" %%a in ('findstr /i "^repo_url:" "%%f"') do (
        set "repo_url=%%b"
        set "repo_url=!repo_url:~1!"
    )
    for /f "tokens=1,* delims=:" %%a in ('findstr /i "^local_path:" "%%f"') do (
        set "local_path=%%b"
        set "local_path=!local_path:~1!"
    )
    for /f "tokens=1,* delims=:" %%a in ('findstr /i "^branch:" "%%f"') do (
        set "branch=%%b"
        set "branch=!branch:~1!"
    )
    for /f "tokens=1,* delims=:" %%a in ('findstr /i "^version:" "%%f"') do (
        set "version=%%b"
        set "version=!version:~1!"
    )

    if "!name!"=="" (
        echo   Skipping: no name found in config
        continue
    )

    if "!repo_url!"=="" (
        echo   Skipping: no repo_url found for !name!
        continue
    )

    if "!local_path!"=="" (
        set "plugin_path=%PROJECT_ROOT%\skills\plugin\!name!"
    ) else (
        set "plugin_path=%PROJECT_ROOT%\!local_path!"
    )
    
    echo   Plugin: !name! (version: !version!)
    echo   Repo: !repo_url!
    echo   Path: !plugin_path!

    if exist "!plugin_path!\.git" (
        echo   Updating existing plugin...
        if not "!branch!"=="" (
            git -C "!plugin_path!" checkout "!branch!"
            git -C "!plugin_path!" pull origin "!branch!"
        ) else (
            git -C "!plugin_path!" pull
        )
    ) else (
        echo   Cloning new plugin...
        if not "!branch!"=="" (
            git clone --branch "!branch!" "!repo_url!" "!plugin_path!"
        ) else (
            git clone "!repo_url!" "!plugin_path!"
        )
    )

    echo   Done!
    echo.
)

echo All plugins have been built successfully!