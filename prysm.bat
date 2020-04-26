@echo off

REM # Use this script to download the latest Prysm release binary.
REM # Usage: ./prysm.bat PROCESS FLAGS
REM #   PROCESS can be: beacon-chain, validator, slasher
REM #   FLAGS are the flags or arguments passed to the PROCESS.
REM # Downloaded binaries are saved to .\dist
REM # Set USE_PRYSM_VERSION to specify a specific release version.
REM #   Example: set USE_PRYSM_VERSION=v0.3.3& .\prysm.bat beacon-chain

SetLocal EnableDelayedExpansion & REM All variables are set local to this run & expanded at execution time rather than at parse time (tip: echo !output!)

set PRYLABS_SIGNING_KEY=0AE0051D647BA3C1A917AF4072E33E4DF1A5036E

REM Complain if no arguments were provided.
if [%1]==[] (
    echo Use this script to download the latest Prysm release binaries.
    echo Usage: ./prysm.bat PROCESS FLAGS.
    echo PROCESS can be beacon-chain, validator, or slasher.
    echo FLAGS are the flags or arguments passed to the PROCESS.
    echo Downloaded binaries are saved to .\dist
    echo Set USE_PRYSM_VERSION to specify a specific release version, Example:
    echo set USE_PRYSM_VERSION=v1.0.0-alpha3
    echo to resume using the latest release:
    echo set USE_PRYSM_VERSION=
    exit /b 1
)

REM Get full path to prysm.bat file (excluding filename)
set wrapper_dir=%~dp1dist
reg Query "HKLM\Hardware\Description\System\CentralProcessor\0" | find /i "x86" > NUL && set WinOS=32BIT || set WinOS=64BIT
if %WinOS%==32BIT (
    echo prysm is only supported on 64-bit Operating Systems
    exit /b 1
)
if %WinOS%==64BIT (
    set arch=amd64.exe
    set system=windows
)

mkdir %wrapper_dir%

REM get_prysm_version - Find the latest Prysm version available for download.
for /f %%i in ('curl -s https://prysmaticlabs.com/releases/latest') do set prysm_version=%%i
echo Latest prysm release is %prysm_version%.
IF defined USE_PRYSM_VERSION (
    echo detected variable USE_PRYSM_VERSION=%USE_PRYSM_VERSION%
    set reason=as specified in USE_PRYSM_VERSION
    set prysm_version=%USE_PRYSM_VERSION%
) else (
    set reason=automatically selected latest available release
)
echo Using prysm version %prysm_version%.

set BEACON_CHAIN_REAL=%wrapper_dir%\beacon-chain-%prysm_version%-%system%-%arch%
set VALIDATOR_REAL=%wrapper_dir%\validator-%prysm_version%-%system%-%arch%
set SLASHER_REAL=%wrapper_dir%\slasher-%prysm_version%-%system%-%arch%

if [%1]==[beacon-chain] (
    if exist %BEACON_CHAIN_REAL% (
        Beacon chain is up to date.
    ) else (
        echo Downloading beacon chain %prysm_version% to %VALIDATOR_REAL% %reason%
        curl -L https://prysmaticlabs.com/releases/beacon-chain-%prysm_version%-%system%-%arch% -o %BEACON_CHAIN_REAL%
        curl --silent -L https://prysmaticlabs.com/releases/beacon-chain-%prysm_version%-%system%-%arch%.sha256 -o %wrapper_dir%\beacon-chain-%prysm_version%-%system%-%arch%.sha256
        curl --silent -L https://prysmaticlabs.com/releases/beacon-chain-%prysm_version%-%system%-%arch%.sig -o %wrapper_dir%\beacon-chain-%prysm_version%-%system%-%arch%.sig
    )
)

if [%1]==[validator] (
    if exist %VALIDATOR_REAL% (
        Validator is up to date.
    ) else (
        echo Downloading validator %prysm_version% to %VALIDATOR_REAL% %reason%
        curl -L https://prysmaticlabs.com/releases/validator-%prysm_version%-%system%-%arch% -o %VALIDATOR_REAL%
        curl --silent -L https://prysmaticlabs.com/releases/validator-%prysm_version%-%system%-%arch%.sha256 -o %wrapper_dir%\validator-%prysm_version%-%system%-%arch%.sha256
        curl --silent -L https://prysmaticlabs.com/releases/validator-%prysm_version%-%system%-%arch%.sig -o %wrapper_dir%\validator-%prysm_version%-%system%-%arch%.sig
    )
)

if [%1]==[slasher] (
    if exist %SLASHER_REAL% (
        Slasher is up to date.
    ) else (
        echo Downloading slasher %prysm_version% to %SLASHER_REAL% %reason%
        curl -L https://prysmaticlabs.com/releases/slasher-%prysm_version%-%system%-%arch% -o %SLASHER_REAL%
        curl --silent -L https://prysmaticlabs.com/releases/slasher-%prysm_version%-%system%-%arch%.sha256 -o %wrapper_dir%\slasher-%prysm_version%-%system%-%arch%.sha256
        curl --silent -L https://prysmaticlabs.com/releases/slasher-%prysm_version%-%system%-%arch%.sig -o %wrapper_dir%\slasher-%prysm_version%-%system%-%arch%.sig
    )
)

if [%1]==[beacon-chain] ( set process=%BEACON_CHAIN_REAL%)
if [%1]==[validator] ( set process=%VALIDATOR_REAL%) 
if [%1]==[slasher] ( set process=%SLASHER_REAL%)

set processargs=%*
echo Starting Prysm %processargs%

set "processargs=!processargs:*%1=!" & REM remove process from the list of arguments
%process% %processargs%
pause
exit /b 0

REM Variables are set local to this run
EndLocal