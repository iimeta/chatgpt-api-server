# ChatGPT-API-SERVER

使用 chat 接口转换对外提供API服务，支持多账号轮循，支持并发控制，支持多种模型。

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
