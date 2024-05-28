package service

import (
	"backend/config"
	"backend/modules/chatgpt/model"
	"backend/utility"
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
	corn, err := gcron.AddSingleton(ctx, config.CRONINTERVAL(ctx), RefreshAllSession, "RefreshSession")
	if err != nil {
		panic(err)
	}
	g.Log().Info(ctx, "RefreshAllSession", "corn", corn, "cornInterval", config.CRONINTERVAL(ctx), "注册成功")
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
		status := 0
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
		plan_type := officialSession.Get("accountCheckInfo.plan_type").String()
		if plan_type == "plus" || plan_type == "team" {
			isPlus = 1
		}
		if plan_type == "free" {
			isPlus = 0

		}

		if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "mfa_bypass") || gstr.Contains(detail, "两步验证") {
			g.Log().Error(ctx, "AddAllSession", "账号异常,跳过刷新", email, detail)
			continue
		}

		getSessionUrl := config.CHATPROXY + "/applelogin"
		sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
			"username":      email,
			"password":      password,
			"authkey":       config.AUTHKEY(ctx),
			"refresh_token": refreshToken,
		})
		sessionJson := gjson.New(sessionVar)
		accessToken = sessionJson.Get("accessToken").String()
		refreshToken = sessionJson.Get("refresh_token").String()
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
		plan_type = sessionJson.Get("accountCheckInfo.plan_type").String()
		if plan_type == "plus" || plan_type == "team" {
			isPlus = 1
		}
		if plan_type == "free" {
			isPlus = 0

		}
		status = 1
		cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
			"officialSession": sessionJson.String(),
			"isPlus":          isPlus,
			"status":          status,
		})

		if status == 0 {
			continue
		}

		// 添加到缓存
		cacheSession := &config.CacheSession{
			Email:        email,
			IsPlus:       isPlus,
			AccessToken:  accessToken,
			CooldownTime: 0,
			RefreshToken: refreshToken,
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

// RefreshAllSession 刷新所有session
func RefreshAllSession(ctx g.Ctx) {
	record, err := cool.DBM(model.NewChatgptSession()).OrderAsc("updateTime").All()
	if err != nil {
		g.Log().Error(ctx, "AddAllSession", err)
		return
	}
	for _, v := range record {
		email := v["email"].String()
		password := v["password"].String()
		isPlus := 0
		status := 0

		officialSession := gjson.New(v["officialSession"])
		refreshToken := officialSession.Get("refresh_token").String()
		detail := officialSession.Get("detail").String()

		getSessionUrl := config.CHATPROXY + "/auth/refresh"
		if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "mfa_bypass") || gstr.Contains(detail, "两步验证") {
			g.Log().Error(ctx, "AddAllSession", "账号异常,跳过刷新", email, detail)
			continue
		}
		if gstr.Contains(detail, "Unknown or invalid refresh token") {
			g.Log().Error(ctx, "AddAllSession", "refreshToken过期,重新登录", email, detail)
			refreshToken = ""
			getSessionUrl = config.CHATPROXY + "/applelogin"
		}

		sessionVar := g.Client().PostVar(ctx, getSessionUrl, g.Map{
			"username":      email,
			"password":      password,
			"refresh_token": refreshToken,
		})
		sessionJson := gjson.New(sessionVar)
		accessToken := sessionJson.Get("accessToken").String()
		if accessToken == "" {
			g.Log().Error(ctx, "AddAllSession", email, "get session error", sessionJson)
			detail := sessionJson.Get("detail").String()
			if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "mfa_bypass") || gstr.Contains(detail, "两步验证") {
				g.Log().Error(ctx, "AddAllSession", email, detail)
				cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
					"officialSession": sessionJson.String(),
					"status":          0,
				})
			}
			continue
		}
		plan_type := sessionJson.Get("accountCheckInfo.plan_type").String()
		if plan_type == "plus" || plan_type == "team" {
			isPlus = 1
		}
		if plan_type == "free" {
			isPlus = 0
		}
		status = 1
		cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
			"officialSession": sessionJson.String(),
			"isPlus":          isPlus,
			"status":          status,
		})

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
