#!/bin/bash

set -e
set -x

go build    -o ./temp/linux_amd64/main main.go 
gf docker main.go -t xyhelper/chatgpt-api-server:yq

# 推送镜像到docker hub
docker push xyhelper/chatgpt-api-server:yq
