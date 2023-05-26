package service

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

type ChatgptSessionService struct {
	*cool.Service
}

func NewChatgptSessionService() *ChatgptSessionService {
	return &ChatgptSessionService{
		&cool.Service{
			Model: model.NewChatgptSession(),
		},
	}
}

// ModifyAfter 新增/删除/修改之后的操作
func (s *ChatgptSessionService) ModifyAfter(ctx g.Ctx, method string, param map[string]interface{}) (err error) {
	g.Log().Debug(ctx, "ChatgptSessionService.ModifyAfter", method, param)
	// 新增/修改 之后，更新session
	if method != "Add" && method != "Update" {
		return
	}
	// 如果没有officialSession，就去获取
	if param["officialSession"] == "" || param["officialSession"] == nil {
		g.Log().Debug(ctx, "ChatgptSessionService.ModifyAfter", "officialSession is empty")
		getSessionUrl := config.CHATPROXY(ctx) + "/getsession"
		g.Client().SetHeader()
		sessionVar := g.Client().SetHeader("authkey",config.AUTHKEY().PostVar(ctx, getSessionUrl, g.Map{
			"username": param["email"],
			"password": param["password"],
		})
		sessionJson := gjson.New(sessionVar)
		if sessionJson.Get("accessToken").String() == "" {
			g.Log().Error(ctx, "ChatgptSessionService.ModifyAfter", "get session error", sessionJson)
			err = gerror.New("get session error")
			return
		}
		_, err = cool.DBM(s.Model).Where("email=?", param["email"]).Update(g.Map{
			"officialSession": sessionJson.String(),
		})
		return
	}
	return
}
