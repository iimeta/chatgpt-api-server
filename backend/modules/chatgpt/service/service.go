package service

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/utility"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
)

func init() {
	ctx := gctx.GetInitCtx()
	go AddAllSession(ctx)
	corn, err := gcron.AddSingleton(ctx, config.CRONINTERVAL(ctx), AddAllSession, "RefreshSession")
	if err != nil {
		panic(err)
	}
	g.Log().Info(ctx, "RefreshSession", "corn", corn, "cornInterval", config.CRONINTERVAL(ctx), "注册成功")
}

// 启动时添加所有账号的session到缓存及set
func AddAllSession(ctx g.Ctx) {
	record, err := cool.DBM(model.NewChatgptSession()).OrderAsc("updateTime").All()
	if err != nil {
		g.Log().Error(ctx, "AddAllSession", err)
		return
	}
	for _, v := range record {
		email := v["email"].String()
		password := v["password"].String()
		isPlus := v["isPlus"].Int()
		status := v["status"].Int()
		officialSession := gjson.New(v["officialSession"])
		accessToken := officialSession.Get("accessToken").String()
		refreshToken := officialSession.Get("refresh_token").String()
		detail := officialSession.Get("detail").String()
		models := officialSession.Get("models").Array()
		if len(models) > 1 {
			isPlus = 1
		} else {
			isPlus = 0
		}
		// 检测accessToken 是否过期,如果过期，就刷新
		err := utility.CheckAccessToken(accessToken)
		if err != nil || status == 0 {
			g.Log().Error(ctx, "AddAllSession", email, err)
			if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "403 Forbidden|Unknown or invalid refresh token.") {
				g.Log().Error(ctx, "AddAllSession", "账号异常,跳过刷新", email, detail)
				continue
			}

			getSessionUrl := config.CHATPROXY(ctx) + "/applelogin"
			sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
				"username":      email,
				"password":      password,
				"authkey":       config.AUTHKEY(ctx),
				"refresh_token": refreshToken,
			})
			sessionJson := gjson.New(sessionVar)
			accessToken = sessionJson.Get("accessToken").String()
			if accessToken == "" {
				g.Log().Error(ctx, "AddAllSession", email, "get session error", sessionJson)
				detail := sessionJson.Get("detail").String()
				if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "403 Forbidden|Unknown or invalid refresh token.") {
					g.Log().Error(ctx, "AddAllSession", email, detail)
					cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
						"officialSession": sessionJson.String(),
						"status":          0,
					})
				}
				continue
			}
			models := sessionJson.Get("models").Array()
			if len(models) > 1 {
				isPlus = 1
			} else {
				isPlus = 0
			}
			status = 1
			cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
				"officialSession": sessionJson.String(),
				"isPlus":          isPlus,
				"status":          status,
			})
		}
		if status == 0 {
			continue
		}

		// 添加到缓存
		cacheSession := &config.CacheSession{
			Email:        email,
			IsPlus:       isPlus,
			AccessToken:  accessToken,
			CooldownTime: 0,
		}
		err = cool.CacheManager.Set(ctx, "session:"+email, cacheSession, time.Hour*24*10)

		if err != nil {
			g.Log().Error(ctx, "AddAllSession to cache ", email, err)
			continue
		}
		g.Log().Info(ctx, "AddAllSession to cache", email, "success")

		// 添加到set
		if isPlus == 1 {
			config.PlusSet.Add(email)
			config.NormalSet.Remove(email)

		} else {
			config.NormalSet.Add(email)
			config.PlusSet.Remove(email)

		}
		accounts_info := officialSession.Get("accounts_info").String()

		teamIds := utility.GetTeamIdByAccountInfo(ctx, accounts_info)
		for _, v := range teamIds {
			config.PlusSet.Add(email + "|" + v)
		}
	}

	g.Log().Info(ctx, "AddSession finish", "plusSet", config.PlusSet.Size(), "normalSet", config.NormalSet.Size())

}
