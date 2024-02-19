#!/bin/bash
set -e

if [ ! -d "./backend/resource/public/xyhelper" ]; then
    echo "Create directory ./backend/resource/public/xyhelper"
    mkdir -p "./backend/resource/public/xyhelper"

    cd frontend
    yarn build
    cd ..
fi

cd backend
gf build main.go -a amd64 -s linux -p ./temp
gf docker main.go -p -t xyhelper/chatgpt-api-server:latest
now=$(date +"%Y%m%d%H%M%S")
# 以当前时间为版本
docker tag xyhelper/chatgpt-api-server:latest xyhelper/chatgpt-api-server:$now
docker push xyhelper/chatgpt-api-server:$now
echo "release success" $now
# 写入发布日志 release.log
echo $now >> ../release.log
