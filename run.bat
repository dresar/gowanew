@echo off
title GOWA - Unified WhatsApp Gateway & UI
echo ========================================================
echo   Starting GOWA Unified Gateway (Backend + Glass UI)
echo ========================================================
echo   Endpoint: http://localhost:3000
echo ========================================================
cd /d "%~dp0src"
go run -tags purego . rest
pause
