@echo off
chcp 65001 >nul
"%~dp0yod.exe" הרץ "%~dp0שרת.יוד"
if errorlevel 1 pause
