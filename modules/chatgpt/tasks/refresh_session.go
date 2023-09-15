package tasks

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/modules/chatgpt/service"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/container/gvar"
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
	result, err := cool.DBM(m).OrderAsc("updateTime").All()
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	for _, v := range result {
		g.Log().Info(ctx, "RefreshSession", v["email"], "start")
		getSessionUrl := "https://chatlogin.xyhelper.cn/getsession"
		var sessionJson *gjson.Json
		// 获取session 最多尝试3次
		for i := 0; i < 3; i++ {
			sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
				"username": v["email"],
				"password": v["password"],
				"authkey":  config.AUTHKEY(ctx),
			})
			sessionJson = gjson.New(sessionVar)
			if sessionJson.Get("accessToken").String() == "" {
				g.Log().Error(ctx, "RefreshSession", v["email"], "get session error", sessionVar)
				continue
			} else {
				break
			}
		}
		if sessionJson.Get("accessToken").String() == "" {
			g.Log().Error(ctx, "RefreshSession", v["email"], "get session error", sessionJson)
			continue
		}
		_, err = cool.DBM(m).Where("email=?", v["email"]).Update(g.Map{
			"officialSession": sessionJson.String(),
			"status":          1,
		})
		if err != nil {
			g.Log().Error(ctx, "RefreshSession", err)
			continue
		}
		// 删除sessionPair
		// delete(service.SessionMap, v["email"].String())
		config.TokenCache.Set(ctx, v["email"].String(), sessionJson.Get("accessToken").String(), time.Hour*24*14)
		g.Log().Info(ctx, "RefreshSession", v["email"], "success")
		// 延时1分钟
		time.Sleep(1 * time.Minute)
	}

}

func OnStartRefreshSession(ctx g.Ctx) {

	m := model.NewChatgptSession()
	result, err := cool.DBM(m).OrderAsc("updateTime").Where("status=0").All()
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	for _, v := range result {
		g.Log().Info(ctx, "RefreshSession", v["email"], "start")
		getSessionUrl := "https://chatlogin.xyhelper.cn/getsession"
		var sessionJson *gjson.Json
		var sessionVar *gvar.Var
		// 获取session 最多尝试3次
		for i := 0; i < 3; i++ {
			sessionVar = g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
				"username": v["email"],
				"password": v["password"],
				"authkey":  config.AUTHKEY(ctx),
			})
			if sessionVar.String() == "密码不正确" {
				g.Log().Error(ctx, "RefreshSession", v["email"], "密码不正确", sessionVar)
				break
			}
			sessionJson = gjson.New(sessionVar)
			if sessionJson.Get("accessToken").String() == "" {
				g.Log().Error(ctx, "RefreshSession", v["email"], "获取accessToken失败", sessionVar)
				continue
			} else {
				break
			}
		}
		if sessionJson.Get("accessToken").String() == "" {
			g.Log().Error(ctx, "RefreshSession", v["email"], "get session error 2", sessionVar)
			_, err = cool.DBM(m).Where("email=?", v["email"]).Update(g.Map{
				"remark": sessionVar.String(),
			})
			if err != nil {
				g.Log().Error(ctx, "RefreshSession", err)
				continue
			}
			continue
		}
		_, err = cool.DBM(m).Where("email=?", v["email"]).Update(g.Map{
			"officialSession": sessionJson.String(),
			"status":          1,
			"remark":          "",
		})
		if err != nil {
			g.Log().Error(ctx, "RefreshSession", err)
			continue
		}
		// 删除sessionPair
		// delete(service.SessionMap, v["email"].String())
		config.TokenCache.Set(ctx, v["email"].String(), sessionJson.Get("accessToken").String(), time.Hour*24*14)
		service.SessionQueue.Push(v["email"].String())
		g.Log().Info(ctx, "RefreshSession", v["email"], "success")
		// 延时1分钟
		// time.Sleep(1 * time.Minute)
	}

}
