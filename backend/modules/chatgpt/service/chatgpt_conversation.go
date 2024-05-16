package service

import (
	"backend/modules/chatgpt/model"

	"github.com/cool-team-official/cool-admin-go/cool"
)

type ChatgptConversationService struct {
	*cool.Service
}

func NewChatgptConversationService() *ChatgptConversationService {
	return &ChatgptConversationService{
		&cool.Service{
			Model: model.NewChatgptConversation(),
		},
	}
}
