package tasks

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/modules/chatgpt/service"
	"chatgpt-api-server/utility"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
)

func init() {
	ctx := gctx.GetInitCtx()
	// 任务注册
	corn, err := gcron.AddSingleton(ctx, config.CRONINTERVAL(ctx), RefreshSession, "RefreshSession")
	if err != nil {
		panic(err)
	}
	g.Log().Info(ctx, "RefreshSession", "corn", corn, "cornInterval", config.CRONINTERVAL(ctx), "注册成功")
	go func() {
		// 延时1分钟
		OnStartRefreshSession(ctx)
	}()
}

func RefreshSession(ctx g.Ctx) {

	m := model.NewChatgptSession()
	result, err := cool.DBM(m).Where("status=0").OrderAsc("updateTime").All()
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	for _, v := range result {
		time.Sleep(time.Second)

		g.Log().Info(ctx, "RefreshSession", v["email"], "start")
		getSessionUrl := config.CHATPROXY(ctx) + "/getsession"
		refreshCookie := gjson.New(v["officialSession"]).Get("refreshCookie").String()
		sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
			"username":      v["email"],
			"password":      v["password"],
			"authkey":       config.AUTHKEY(ctx),
			"refreshCookie": refreshCookie,
		})
		sessionJson := gjson.New(sessionVar)
		accessToken := sessionJson.Get("accessToken").String()
		// sessionJson.Dump()
		if accessToken == "" {
			g.Log().Error(ctx, "RefreshSession", v["email"], "get session error", sessionJson)
			detail := sessionJson.Get("detail").String()
			if detail != "" {
				cool.DBM(model.NewChatgptSession()).Where("email", v["email"]).Update(g.Map{"status": 0, "remark": detail})
			}
			continue
		}
		if err := utility.CheckAccessToken(accessToken); err != nil {
			g.Log().Error(ctx, "RefreshSession", v["email"], "check token error", err)
			cool.DBM(model.NewChatgptSession()).Where("email", v["email"]).Update(g.Map{"status": 0, "remark": err.Error()})
			continue
		}
		IsPlusAccount := 0

		models := sessionJson.GetJson("models")

		// g.DumpWithType(models)
		if len(models.Array()) > 1 {
			IsPlusAccount = 1
		}
		_, err = cool.DBM(m).Where("email=?", v["email"]).Update(g.Map{
			"officialSession": sessionJson.String(),
			"status":          1,
			"remark":          "",
			"isPlus":          IsPlusAccount,
		})
		if err != nil {
			g.Log().Error(ctx, "RefreshSession", err)
			continue
		}
		if IsPlusAccount == 1 {
			service.SessionQueue.Push(v["email"].String())
			config.TokenCache.Set(ctx, v["email"].String(), sessionJson.Get("accessToken").String(), 0)

			g.Log().Info(ctx, "RefreshSession", v["email"], "success")
			g.Log().Info(ctx, "~~~~~~~~~~~~~~~RefreshSession", v["email"], "success")
		} else {
			g.Log().Info(ctx, "RefreshSession", v["email"], "not plus")
		}

	}

}

func OnStartRefreshSession(ctx g.Ctx) {

	m := model.NewChatgptSession()
	result, err := cool.DBM(m).OrderAsc("updateTime").All()
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	for _, v := range result {
		time.Sleep(time.Second)

		g.Log().Info(ctx, "RefreshSession", v["email"], "start")
		getSessionUrl := config.CHATPROXY(ctx) + "/getsession"
		refreshCookie := gjson.New(v["officialSession"]).Get("refreshCookie").String()
		sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
			"username":      v["email"],
			"password":      v["password"],
			"authkey":       config.AUTHKEY(ctx),
			"refreshCookie": refreshCookie,
		})
		sessionJson := gjson.New(sessionVar)
		// sessionJson.Dump()
		if sessionJson.Get("accessToken").String() == "" {
			g.Log().Error(ctx, "RefreshSession", v["email"], "get session error", sessionJson)
			detail := sessionJson.Get("detail").String()
			if detail != "" {
				cool.DBM(model.NewChatgptSession()).Where("email", v["email"]).Update(g.Map{"status": 0, "remark": detail})
			}
			continue
		}
		IsPlusAccount := 0

		models := sessionJson.GetJson("models")

		// g.DumpWithType(models)
		if len(models.Array()) > 1 {
			IsPlusAccount = 1
		}
		_, err = cool.DBM(m).Where("email=?", v["email"]).Update(g.Map{
			"officialSession": sessionJson.String(),
			"status":          1,
			"remark":          "",
			"isPlus":          IsPlusAccount,
		})
		if err != nil {
			g.Log().Error(ctx, "RefreshSession", err)
			continue
		}
		if IsPlusAccount == 1 {
			service.SessionQueue.Push(v["email"].String())
			config.TokenCache.Set(ctx, v["email"].String(), sessionJson.Get("accessToken").String(), 0)

			g.Log().Info(ctx, "RefreshSession", v["email"], "success")
			g.Log().Info(ctx, "~~~~~~~~~~~~~~~RefreshSession", v["email"], "success")
		} else {
			g.Log().Info(ctx, "RefreshSession", v["email"], "not plus")
		}

	}

}
