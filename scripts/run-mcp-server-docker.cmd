@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-mcp-server-docker.ps1" %*
