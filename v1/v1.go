package v1

import (
	"chatgpt-api-server/v1/chat"

	"github.com/gogf/gf/v2/frame/g"
)

func init() {
	s := g.Server()
	v1Group := s.Group("/v1")
	v1Group.ALL("/chat/completions", chat.Completions)
	v1Group.ALL("/chat/gpt4v", chat.Gpt4v)
}
