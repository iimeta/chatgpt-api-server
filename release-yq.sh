#!/bin/bash

set -e

gf build main.go -a amd64 -s linux
gf docker main.go -t xyhelper/chatgpt-api-server:yq

# 推送镜像到docker hub
docker push xyhelper/chatgpt-api-server:yq
