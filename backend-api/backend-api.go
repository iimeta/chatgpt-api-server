package backendapi

import (
	"chatgpt-api-server/modules/chatgpt/service"

	"github.com/gogf/gf/v2/frame/g"
)

var (
	ChatgptUserService = service.NewChatgptUserService()
)

func init() {
	// 注册路由
	s := g.Server()
	backendApiGroup := s.Group("/backend-api")
	backendApiGroup.POST("/conversation", Conversation)

}
