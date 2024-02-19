package service

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/utility"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

type ChatgptSessionService struct {
	*cool.Service
}

func NewChatgptSessionService() *ChatgptSessionService {
	return &ChatgptSessionService{
		&cool.Service{
			Model: model.NewChatgptSession(),
			UniqueKey: g.MapStrStr{
				"email": "邮箱不能重复",
			},
			NotNullKey: g.MapStrStr{
				"email":    "邮箱不能为空",
				"password": "密码不能为空",
			},
			PageQueryOp: &cool.QueryOp{
				FieldEQ:      []string{"email", "password", "officialSession", "remark"},
				KeyWordField: []string{"email", "password", "officialSession", "remark"},
			},
		},
	}
}

// MofifyBefore 新增/删除/修改之前的操作
func (s *ChatgptSessionService) ModifyBefore(ctx g.Ctx, method string, param map[string]interface{}) (err error) {
	g.Log().Debug(ctx, "ChatgptSessionService.ModifyBefore", method, param)

	// g.Dump(idsJson)
	// 如果是删除，就删除缓存及set
	if method == "Delete" {
		ids := gjson.New(param["ids"]).Array()
		for _, id := range ids {
			record, err := cool.DBM(s.Model).Where("id=?", id).One()
			if err != nil {
				return err
			}
			email := record["email"].String()
			isPlus := record["isPlus"].Int()

			// 删除缓存
			cool.CacheManager.Remove(ctx, "session:"+email)
			// 删除set
			if isPlus == 1 {
				config.PlusSet.Remove(email)
			} else {
				config.NormalSet.Remove(email)
			}
		}
	}

	return
}

// ModifyAfter 新增/删除/修改之后的操作
func (s *ChatgptSessionService) ModifyAfter(ctx g.Ctx, method string, param map[string]interface{}) (err error) {
	g.Log().Debug(ctx, "ChatgptSessionService.ModifyAfter", method, param)
	// 新增/修改 之后，更新session
	if method != "Add" && method != "Update" {
		return
	}
	officialSession := gjson.New(param["officialSession"])
	refreshToken := officialSession.Get("refresh_token").String()

	// 如果没有officialSession，就去获取
	s.GetSessionAndUpdateStatus(ctx, param, refreshToken)
	return
}

// // GetSessionByUserToken 根据userToken获取session
// func (s *ChatgptSessionService) GetSessionByUserToken(ctx g.Ctx, userToken string, conversationId string, isPlusModel bool) (record gdb.Record, code int, err error) {
// 	if conversationId != "" {
// 		rec, err := cool.DBM(model.NewChatgptConversation()).Where(g.Map{
// 			"conversationId": conversationId,
// 			"userToken":      userToken,
// 		}).One()
// 		if err != nil {
// 			return nil, 500, err
// 		}
// 		if rec.IsEmpty() {
// 			return nil, 404, nil
// 		}
// 		email := rec["email"].String()
// 		record, err = cool.DBM(s.Model).Where("email=?", email).One()
// 		if err != nil {
// 			return nil, 500, err
// 		}
// 		if record.IsEmpty() {
// 			return nil, 404, nil
// 		}
// 		return record, 200, err
// 	}
// 	// officalSession不为空
// 	m := cool.DBM(s.Model).Where("status=1").Where("officialSession != ''")
// 	g.Log().Debug(ctx, "ChatgptSessionService.GetSessionByUserToken", "isPlusModel", isPlusModel)
// 	if isPlusModel {
// 		m = m.Where("isPlus", 1)
// 	} else {
// 		m = m.Where("isPlus", 0)
// 	}
// 	record, err = m.OrderRandom().One()
// 	if err != nil {
// 		return nil, 500, err
// 	}
// 	if record.IsEmpty() {
// 		err = gerror.New("无可用session")

// 		return nil, 501, err
// 	}

// 	return record, 200, err
// }

func (s *ChatgptSessionService) GetSessionAndUpdateStatus(ctx g.Ctx, param g.Map, refreshToken string) error {
	getSessionUrl := config.CHATPROXY(ctx) + "/applelogin"
	sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).SetCookie("arkoseToken", gconv.String(param["arkoseToken"])).PostVar(ctx, getSessionUrl, g.Map{
		"username":      param["email"],
		"password":      param["password"],
		"authkey":       config.AUTHKEY(ctx),
		"refresh_token": refreshToken,
	})
	sessionJson := gjson.New(sessionVar)
	if sessionJson.Get("accessToken").String() == "" {
		g.Log().Error(ctx, "ChatgptSessionService.ModifyAfter", "get session error", sessionJson)
		detail := sessionJson.Get("detail").String()
		if detail != "" {
			err := gerror.New(detail)
			cool.DBM(s.Model).Where("email=?", param["email"]).Update(g.Map{
				"officialSession": sessionJson.String(),
				"status":          0,
			})
			return err
		} else {
			return gerror.New("get session error")
		}
	}
	var isPlus int
	models := sessionJson.Get("models").Array()
	if len(models) > 1 {
		isPlus = 1
	} else {
		isPlus = 0
	}
	_, err := cool.DBM(s.Model).Where("email=?", param["email"]).Update(g.Map{
		"officialSession": sessionJson.String(),
		"isPlus":          isPlus,
		"status":          1,
	})
	// 写入缓存及set
	email := param["email"].(string)
	cacheSession := &config.CacheSession{
		Email:       email,
		AccessToken: sessionJson.Get("accessToken").String(),
		IsPlus:      isPlus,
	}
	cool.CacheManager.Set(ctx, "session:"+email, cacheSession, time.Hour*24*10)
	// 添加到set
	if isPlus == 1 {
		config.PlusSet.Add(email)
		config.NormalSet.Remove(email)

	} else {
		config.NormalSet.Add(email)
		config.PlusSet.Remove(email)
	}
	accounts_info := sessionJson.Get("accounts_info").String()

	teamIds := utility.GetTeamIdByAccountInfo(ctx, accounts_info)
	for _, v := range teamIds {
		config.PlusSet.Add(email + "|" + v)
	}
	g.Log().Info(ctx, "AddSession finish", "plusSet", config.PlusSet.Size(), "normalSet", config.NormalSet.Size())

	return err
}
