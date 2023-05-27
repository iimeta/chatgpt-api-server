#!/bin/bash

set -e


gf docker main.go -t xyhelper/chatgpt-api-server:latest
# 修改镜像标签为当前日期时间
time=$(date "+%Y%m%d%H%M%S")
docker tag xyhelper/chatgpt-api-server:latest xyhelper/chatgpt-api-server:$time
# 推送镜像到docker hub
docker push xyhelper/chatgpt-api-server:latest
docker push xyhelper/chatgpt-api-server:$time