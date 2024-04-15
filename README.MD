# ChatGPT-API-SERVER
## 本服务为商业服务，自2024年4月16日0:00不再提供免费接入点。需付费使用。

1.流量大使用付费接入点或者小流量拼车付费接入点的，可打开以下链接选择适合自己的方案
https://xyhelper.cn/access

2.流量小或者小团队自用的，可使用蟑螂v1，目前蟑螂v1可正常使用，了解详情请访问:
CockroachAi（又名蟑螂）
https://github.com/cockroachai/cockroachai

## 使用 chat 接口转换对外提供API服务，支持多账号轮循，支持并发控制，支持多种模型。

快速部署脚本：[ChatGPT-API-SERVER-Deploy](https://github.com/xyhelper/chatgpt-api-server-deploy/)。

## 环境变量

```yaml
# 接入网关地址
CHATPROXY: "https://demo.xyhelper.cn"
# 接入网关的authkey
AUTHKEY: "xyhelper"
# 是否限制每个用户只能同时发起一个会话
USERTOKENLOCK: true
# 3.5模型是否免费
ISFREE: true
```
