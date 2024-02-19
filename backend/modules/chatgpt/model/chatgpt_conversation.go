package model

import (
	"github.com/cool-team-official/cool-admin-go/cool"
)

const TableNameChatgptConversation = "chatgpt_conversation"

// ChatgptConversation mapped from table <chatgpt_conversation>
type ChatgptConversation struct {
	*cool.Model
	UserToken      string `gorm:"index;column:userToken;not null;comment:用户Token" json:"userToken"`
	Email          string `gorm:"column:email;not null;comment:邮箱" json:"email"`
	ConversationId string `gorm:"column:conversationId;not null;comment:会话ID" json:"conversationId"`
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
