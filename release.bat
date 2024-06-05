@echo off

REM 获取当前日期和时间，格式为YYYYMMDDHHmmSS
for /f "tokens=1-4 delims=/ " %%a in ("%date%") do (
    set YEAR=%%a
    set MONTH=%%b
    set DAY=%%c
)
for /f "tokens=1-3 delims=:." %%a in ("%time%") do (
    set HOUR=%%a
    set MINUTE=%%b
    set SECOND=%%c
)

REM 如果HOUR小于10，前面补0
if %HOUR% lss 10 (
    set HOUR=0%HOUR%
)

REM 拼接版本号
set VERSION=%YEAR%%MONTH%%DAY%%HOUR%%MINUTE%%SECOND%

set IMAGE_BASE=liuhuapiaoyuan/tripo-api


REM 构建前端、移动端和后端的 Docker 镜像
docker build -t %IMAGE_BASE%-client:%VERSION% .

REM 推送镜像到 Docker Hub
docker push %IMAGE_BASE%-client:%VERSION%
docker tag %IMAGE_BASE%-client:%VERSION% %IMAGE_BASE%-client:latest
docker push %IMAGE_BASE%-client:latest


@REM echo 发布结果 %VERSION%
echo version publish success:  %IMAGE_BASE%-client:%VERSION%