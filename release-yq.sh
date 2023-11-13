#!/bin/bash

set -e
set -x

gf build main.go -a amd64 -s linux
gf docker main.go -t xyhelper/chatgpt-api-server:35

# 推送镜像到docker hub
docker push xyhelper/chatgpt-api-server:35
