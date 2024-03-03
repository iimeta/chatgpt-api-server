package v1

import (
	"chatgpt-api-server/v1/chat"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func init() {
	s := g.Server()
	v1Group := s.Group("/v1")
	v1Group.Middleware(MiddlewareCORS)
	v1Group.POST("/chat/completions", chat.Completions)
	v1Group.POST("/chat/gpt4v", chat.Gpt4v)
	v1Group.POST("/chat/gpt4v-mobile", chat.Gpt4v)

}

func MiddlewareCORS(r *ghttp.Request) {
	r.Response.CORSDefault()
	r.Middleware.Next()
}
