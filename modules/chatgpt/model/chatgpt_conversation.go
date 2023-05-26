package model

import (
	"github.com/cool-team-official/cool-admin-go/cool"
)

const TableNameChatgptConversation = "chatgpt_conversation"

// ChatgptConversation mapped from table <chatgpt_conversation>
type ChatgptConversation struct {
	*cool.Model
	User           string `gorm:"index;column:user;not null;comment:用户" json:"user"`
	ConversationId string `gorm:"index;column:conversationId;not null;comment:conversationId" json:"conversationId"`
	TokenId        int    `gorm:"index;column:tokenId;not null;comment:tokenId" json:"tokenId"`
}

// TableName ChatgptConversation's table name
func (*ChatgptConversation) TableName() string {
	return TableNameChatgptConversation
}

// GroupName ChatgptConversation's table group
func (*ChatgptConversation) GroupName() string {
	return "default"
}

// NewChatgptConversation create a new ChatgptConversation
func NewChatgptConversation() *ChatgptConversation {
	return &ChatgptConversation{
		Model: cool.NewModel(),
	}
}

// init 创建表
func init() {
	cool.CreateTable(&ChatgptConversation{})
}
