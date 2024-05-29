package chat

import (
	backendapi "backend/backend-api"
	"backend/config"
	"backend/modules/chatgpt/model"
	"strings"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/launchdarkly/eventsource"
)

func Completions(r *ghttp.Request) {
	ctx := r.Context()
	// 获取 Header 中的 Authorization	去除 Bearer
	userToken := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")
	// 如果 Authorization 为空，返回 401
	if userToken == "" {
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"detail": "Authentication credentials were not provided.",
		})
	}
	isPlusUser := false
	if !config.ISFREE(ctx) {
		userRecord, err := cool.DBM(model.NewChatgptUser()).Where("userToken", userToken).Where("expireTime>now()").Cache(gdb.CacheOption{
			Duration: 10 * time.Minute,
			Name:     "userToken:" + userToken,
			Force:    true,
		}).One()
		if err != nil {
			g.Log().Error(ctx, err)
			r.Response.Status = 500
			r.Response.WriteJson(g.Map{
				"detail": err.Error(),
			})
			return
		}
		if userRecord.IsEmpty() {
			g.Log().Error(ctx, "userToken not found:", userToken)
			r.Response.Status = 401
			r.Response.WriteJson(g.Map{
				"detail": "userToken not found",
			})
			return
		}
		if userRecord["isPlus"].Int() == 1 {
			isPlusUser = true
		}
	}

	// g.Log().Debug(ctx, "token: ", token)
	reqJson, err := r.GetJson()
	if err != nil {
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"error": "bad request",
		})
	}
	reqModel := reqJson.Get("model").String()
	realModel := config.GetModel(ctx, reqModel)
	reqJson.Set("model", realModel)
	// g.Dump(req)

	// 如果不是plus用户但是使用了plus模型
	if !isPlusUser && gstr.HasPrefix(realModel, "gpt-4") {
		r.Response.Status = 501
		r.Response.WriteJson(g.Map{
			"detail": "plus user only",
		})
		return
	}
	email := ""
	teamId := ""
	emailWithTeamId := ""
	clears_in := 0
	ok := false
	// plus失效
	isPlusInvalid := false
	// 是否归还
	isReturn := true
	isPlusModel := gstr.HasPrefix(realModel, "gpt-4")
	if isPlusModel {
		defer func() {
			go func() {
				if email != "" && isReturn {
					if isPlusInvalid {
						// 如果plus失效，将isPlus设置为0
						cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
							"isPlus": 0,
						})
						// 从set中删除
						config.PlusSet.Remove(emailWithTeamId)
						// 添加到set
						config.NormalSet.Add(email)
						g.Log().Info(ctx, "PLUS失效归还", email, "添加到NormalSet")
						return
					}
					if clears_in > 0 {
						// 延迟归还
						g.Log().Info(ctx, "延迟"+gconv.String(clears_in)+"秒归还", emailWithTeamId, "到PlusSet")

						time.Sleep(time.Duration(clears_in) * time.Second)

					}
					config.PlusSet.Add(emailWithTeamId)
					g.Log().Info(ctx, "归还", emailWithTeamId, "到PlusSet")
				}
			}()
		}()
		if email == "" {
			emailWithTeamId, ok = config.PlusSet.Pop()
			g.Log().Info(ctx, emailWithTeamId, ok)
			if !ok {
				g.Log().Error(ctx, "Get email from set error")
				r.Response.Status = 502
				r.Response.WriteJson(g.Map{
					"detail": "Server is busy, please try again later|502",
				})
				return
			}
			if gstr.Contains(emailWithTeamId, "|") {
				emailWithTeamIdArr := gstr.Split(emailWithTeamId, "|")
				email = emailWithTeamIdArr[0]
				teamId = emailWithTeamIdArr[1]
			} else {
				email = emailWithTeamId
			}
		}
	} else {
		emailWithTeamId, ok = config.NormalSet.Pop()
		if !ok {
			g.Log().Error(ctx, "Get email from set error")
			r.Response.Status = 500
			r.Response.WriteJson(g.Map{
				"detail": "Server is busy, please try again later",
			})
			return
		}
		defer func() {
			go func() {
				if email != "" && isReturn {
					config.NormalSet.Add(email)
				}
			}()
		}()

		email = emailWithTeamId
	}
	if email == "" {
		g.Log().Error(ctx, "Get email from set error")
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"detail": "Server is busy, please try again later",
		})
		return
	}
	g.Log().Info(ctx, userToken, "使用", emailWithTeamId, reqModel, "->", realModel, "发起会话")

	// 使用email获取 accessToken
	sessionCache := &config.CacheSession{}
	cool.CacheManager.MustGet(ctx, "session:"+email).Scan(&sessionCache)

	// ChatReq.Dump()
	// 请求openai
	reqHeader := g.MapStrStr{
		"Authorization":     "Bearer " + sessionCache.RefreshToken,
		"Content-Type":      "application/json",
		"Replay-Real-Model": "true",
	}
	if teamId != "" {
		reqHeader["ChatGPT-Account-ID"] = teamId
	}
	resp, err := g.Client().SetHeaderMap(reqHeader).Post(ctx, config.CHATPROXY+"/v1/chat/completions", reqJson)
	if err != nil {
		g.Log().Error(ctx, "g.Client().Post error: ", err)
		r.Response.Status = 500
		r.Response.WriteJson(gjson.New(`{"detail": "internal server error"}`))
		return
	}
	defer resp.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 402 {
		g.Log().Error(ctx, "token过期,需要重新获取token", email, resp.ReadAllString())
		isReturn = false
		cool.DBM(model.NewChatgptSession()).Where("email", email).Update(g.Map{
			"status":          0, // token过期
			"officialSession": "token过期,需要重新获取token",
		})
		isReturn = false
		go backendapi.RefreshSession(email)
		r.Response.WriteStatus(401, resp.ReadAllString())
		return
	}
	if resp.StatusCode == 429 {
		resStr := resp.ReadAllString()

		clears_in = gjson.New(resStr).Get("detail.clears_in").Int()

		if clears_in > 0 {
			g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==429", resStr)

			r.Response.WriteStatusExit(429, resStr)
			return
		} else {
			g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==429", resStr)

			r.Response.WriteStatusExit(429, resStr)
			return
		}
	}
	if resp.StatusCode == 403 {
		contentType := resp.Header.Get("Content-Type")
		resStr := resp.ReadAllString()

		g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==403", contentType, resStr)
		code := gjson.New(resStr).Get("detail.code").String()
		if code != "" {
			isReturn = false
			cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
				"status":          0,
				"officialSession": code,
			})
			go backendapi.RefreshSession(email)
		}

		r.Response.WriteStatusExit(403, resp.ReadAllString())
		return

	}
	// 如果返回结果不是200
	if resp.StatusCode != 200 {
		g.Log().Error(ctx, "resp.StatusCode: ", resp.StatusCode)
		r.Response.Status = resp.StatusCode
		if resp.Header.Get("Content-Type") == "application/json" {
			r.Response.WriteJson(gjson.New(resp.ReadAllString()))
		} else {
			r.Response.WriteJson(g.Map{
				"detail": "openai respone error|" + resp.Status,
			})
		}
		return
	}

	// 流式返回
	if gstr.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		r.Response.Header().Set("Cache-Control", "no-cache")
		r.Response.Header().Set("Connection", "keep-alive")
		modelSlug := ""

		decoder := eventsource.NewDecoder(resp.Body)
		defer decoder.Decode()

		for {
			event, err := decoder.Decode()
			if err != nil {
				// if err == io.EOF {
				// 	break
				// }
				g.Log().Info(ctx, "释放资源")
				break
			}
			text := event.Data()
			// g.Log().Debug(ctx, "text: ", text)
			if text == "" {
				continue
			}
			if text == "[DONE]" {
				r.Response.Writeln("data: " + text + "\n")
				r.Response.Flush()
				continue
			}
			respJson := gjson.New(text)

			replayModel := respJson.Get("model").String()
			if replayModel != "" {
				modelSlug = replayModel
			}
			respJson.Set("model", reqModel)

			r.Response.Writeln("data: " + respJson.String() + "\n")
			r.Response.Flush()
		}

		if realModel != "text-davinci-002-render-sha" && realModel != "auto" && modelSlug == "text-davinci-002-render-sha" {
			isPlusInvalid = true
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}
		r.ExitAll()
		return

	} else {
		respJson := gjson.New(resp.ReadAllString())
		modelSlug := respJson.Get("model").String()
		respJson.Set("model", reqModel)
		r.Response.WriteJson(respJson)
		if realModel != "text-davinci-002-render-sha" && realModel != "auto" && modelSlug == "text-davinci-002-render-sha" {
			isPlusInvalid = true

			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}
		return
	}

}
