package backendapi

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/utility"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/launchdarkly/eventsource"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
)

var (
	continueRequest = `{"action":"continue","conversation_id":"f8cdda28-fcae-4dc8-b8b6-687af2741ee7","parent_message_id":"c22837bf-c1f9-4579-a2b4-71102670cfe2","model":"text-davinci-002-render-sha","timezone_offset_min":-480,"history_and_training_disabled":false}`
)

func Conversation(r *ghttp.Request) {

	ctx := r.GetCtx()

	// 获取 Header 中的 Authorization	去除 Bearer
	userToken := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")
	// 如果 Authorization 为空，返回 401
	if userToken == "" {
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"detail": "Authentication credentials were not provided.",
		})
	}
	reqJson, err := r.GetJson()
	if err != nil {
		g.Log().Error(ctx, err)
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"detail": "unable to parse request body",
		})
		return
	}
	g.Log().Debug(ctx, userToken, reqJson)
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
	reqModel := reqJson.Get("model").String()
	history_and_training_disabled := reqJson.Get("history_and_training_disabled").Bool()
	var isPlusModel bool
	g.Log().Debug(ctx, "reqModel", reqModel, config.PlusModels.ContainsI(reqModel), config.FreeModels.Contains(reqModel))
	g.Log().Debug(ctx, "reqModel", reqModel, config.FreeModels.ContainsI(reqModel), config.PlusModels.Contains(reqModel))
	if config.PlusModels.ContainsI(reqModel) {
		isPlusModel = true
	} else if config.FreeModels.ContainsI(reqModel) {
		isPlusModel = false
	} else {
		reqJson.Set("model", config.DefaultModel)
		isPlusModel = false
	}
	g.Log().Debug(ctx, "isPlusModel", isPlusModel)
	// 如果是plus模型，但是用户不是plus用户，则返回501
	if isPlusModel && !isPlusUser {
		r.Response.Status = 501
		r.Response.WriteJson(g.Map{
			"detail": "userToken is not plus user",
		})
		return
	}
	email := ""
	teamId := ""
	emailWithTeamId := ""
	ok := false

	clears_in := 0
	// plus失效
	isPlusInvalid := false
	// 是否归还
	isReturn := true
	client := g.Client()

	// 如果带有conversation_id，说明是继续会话，需要获取email	并获取accessToken
	conversation_id := reqJson.Get("conversation_id").String()
	if conversation_id != "" {
		emailWithTeamId = cool.CacheManager.MustGet(ctx, "conversation:"+conversation_id).String()
		if emailWithTeamId == "" {
			r.Response.Status = 404
			r.Response.WriteJson(g.Map{
				"detail": "conversation_id not found",
			})
			return
		}
		// 如果emailWithTeamId 包含 | 说明是团队模式 使用|分割为 email 和 teamId
		if gstr.Contains(emailWithTeamId, "|") {
			emailWithTeamIdArr := gstr.Split(emailWithTeamId, "|")
			email = emailWithTeamIdArr[0]
			teamId = emailWithTeamIdArr[1]
		} else {
			email = emailWithTeamId
		}
		// 使用email获取 accessToken
		var sessionCache *config.CacheSession
		cool.CacheManager.MustGet(ctx, "session:"+email).Scan(&sessionCache)
		accessToken := sessionCache.AccessToken
		if accessToken == "" {
			r.Response.Status = 404
			r.Response.WriteJson(g.Map{
				"detail": "accessToken not found",
			})
			return
		}
		client.SetHeader("Authorization", "Bearer "+accessToken)
	} else {
		// 如果不带conversation_id，说明是新会话，需要获取email	并获取accessToken

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
				if !ok {
					g.Log().Error(ctx, "Get email from set error")
					r.Response.Status = 500
					r.Response.WriteJson(g.Map{
						"detail": "Server is busy, please try again later",
					})
					return
				}
				// 如果emailWithTeamId 包含 | 说明是团队模式 使用|分割为 email 和 teamId
				g.Log().Info(ctx, "获取", emailWithTeamId, "从PlusSet")
				if gstr.Contains(emailWithTeamId, "|") {
					emailWithTeamIdArr := gstr.Split(emailWithTeamId, "|")
					email = emailWithTeamIdArr[0]
					teamId = emailWithTeamIdArr[1]
				} else {
					email = emailWithTeamId
				}
			}
		} else {
			emailPop, ok := config.NormalSet.Pop()
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

			email = emailPop
		}
		if email == "" {
			g.Log().Error(ctx, "Get email from set error")
			r.Response.Status = 500
			r.Response.WriteJson(g.Map{
				"detail": "Server is busy, please try again later",
			})
			return
		}
		// 使用email获取 accessToken
		var sessionCache *config.CacheSession
		cool.CacheManager.MustGet(ctx, "session:"+email).Scan(&sessionCache)
		accessToken := sessionCache.AccessToken
		err = utility.CheckAccessToken(accessToken)
		if err != nil { // accessToken失效
			g.Log().Error(ctx, err)
			isReturn = false
			go RefreshSession(email)
			r.Response.Status = 401
			r.Response.WriteJson(g.Map{
				"detail": "accessToken is invalid,will be refresh",
			})
			return
		}

		client.SetHeader("Authorization", "Bearer "+accessToken)
	}

	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("authkey", config.AUTHKEY(ctx))
	if teamId != "" {
		client.SetHeader("ChatGPT-Account-ID", teamId)
	}
	realModel := reqJson.Get("model").String()
	g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", realModel, "发起会话")

	resp, err := client.Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", reqJson)
	if err != nil {
		g.Log().Error(ctx, err)
		r.Response.WriteStatusExit(500)
	}
	defer resp.Close()
	defer resp.Body.Close()
	g.Log().Debug(ctx, resp.StatusCode, resp.Header.Get("Content-Type"))
	// 如果返回401 说明token过期，需要重新获取token 先删除sessionPair 并将status设置为0
	if resp.StatusCode == 401 {
		g.Log().Error(ctx, "token过期,需要重新获取token", email, resp.ReadAllString())
		isReturn = false
		cool.DBM(model.NewChatgptSession()).Where("email", email).Update(g.Map{"status": 0})
		go RefreshSession(email)
		r.Response.WriteStatusExit(401)
		return
	}
	if resp.StatusCode == 429 {
		resStr := resp.ReadAllString()

		clears_in = gjson.New(resStr).Get("detail.clears_in").Int()

		if clears_in > 0 {
			g.Log().Error(ctx, email, "resp.StatusCode==430", resStr)

			r.Response.WriteStatusExit(430, resStr)
			return
		} else {
			g.Log().Error(ctx, email, "resp.StatusCode==429", resStr)

			r.Response.WriteStatusExit(429, resStr)
			return
		}
	}
	if resp.StatusCode != 200 {
		resp.RawDump()
		respText := resp.ReadAllString()
		g.Log().Error(ctx, email, "resp.StatusCode!=200", resp.StatusCode, respText)
		r.Response.WriteStatusExit(resp.StatusCode, respText)
		return
	}

	if resp.StatusCode == 200 && resp.Header.Get("Content-Type") == "text/event-stream; charset=utf-8" {
		r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		r.Response.Header().Set("Cache-Control", "no-cache")
		r.Response.Header().Set("Connection", "keep-alive")
		//  流式回应
		rw := r.Response.RawWriter()
		flusher, ok := rw.(http.Flusher)
		if !ok {
			g.Log().Error(ctx, "rw.(http.Flusher) error")
			r.Response.WriteStatusExit(500)
			return
		}
		messageId := ""
		messagBody := ""
		conversationId := ""
		modelSlug := ""
		streamOption := eventsource.DecoderOptionReadTimeout(600 * time.Second)
		eventsource.NewSliceRepository()
		decoder := eventsource.NewDecoderWithOptions(resp.Body, streamOption)
		finishType := ""
		for {
			event, err := decoder.Decode()
			if err != nil {
				if err == io.EOF {
					break
				}
				// g.Log().Error(ctx, "decoder.Decode error", err)
				break
			}
			text := event.Data()
			if text == "" {
				continue
			}
			if text == "[DONE]" {
				_, err = fmt.Fprint(rw, "data: "+text+"\n\n")
				if err != nil {
					g.Log().Error(ctx, "fmt.Fprintf error", err)
					r.Response.WriteStatusExit(500)
					return
				}
				flusher.Flush()
				continue
			}
			// g.Log().Debug(ctx, "text", gjson.New(text))
			messeage_id := gjson.New(text).Get("message.id").String()
			conversation_id := gjson.New(text).Get("conversation_id").String()
			model_slug := gjson.New(text).Get("message.metadata.model_slug").String()
			finish_type := gjson.New(text).Get("message.metadata.finish_details.type").String()
			message_body := gjson.New(text).Get("message.content.parts.0").String()
			if message_body == "" {
				continue
			}

			// g.Log().Debug(ctx, "conversation_id", conversation_id)
			if conversation_id != "" {
				conversationId = conversation_id
			}
			if model_slug != "" {
				modelSlug = model_slug
			}
			if messeage_id != "" {
				messageId = messeage_id
			}
			if message_body != "" {
				messagBody = message_body
			}
			if finish_type != "" {
				finishType = finish_type
			}
			// r.Response.Writefln("data: %s\n\n", text)
			// r.Response.Flush()
			_, err = fmt.Fprintf(rw, "data: %s\n\n", text)

			if err != nil {
				g.Log().Error(ctx, "fmt.Fprintf error", err)
				resp.Body.Close()
				continue
			}
			flusher.Flush()
		}
		g.Log().Debug(ctx, "finishType", finishType)
		g.Log().Debug(ctx, "conversationId", conversationId)
		g.Log().Debug(ctx, "modelSlug", modelSlug)
		g.Log().Debug(ctx, "messageId", messageId)
		if realModel != "text-davinci-002-render-sha" && modelSlug == "text-davinci-002-render-sha" {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}
		// g.Log().Debug(ctx, "messagBody", messagBody)
		// 如果是max_tokens类型的完成,说明会话未结束，需要继续请求
		count := 0
		for finishType == "max_tokens" && count < config.CONTINUEMAX(ctx) {
			count++

			g.Log().Debug(ctx, "finishType", finishType, "继续请求，count:", count)
			continueJson := gjson.New(continueRequest)
			continueJson.Set("conversation_id", conversationId)
			continueJson.Set("model", modelSlug)
			continueJson.Set("parent_message_id", messageId)
			continueJson.Set("history_and_training_disabled", history_and_training_disabled)
			g.Log().Debug(ctx, "continueJson", continueJson)
			continueresp, err := client.Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", continueJson)
			if err != nil {
				r.Response.WriteStatusExit(500)
			}
			defer continueresp.Close()
			defer continueresp.Body.Close()
			g.Log().Debug(ctx, "continueresp.StatusCode", continueresp.StatusCode)
			if continueresp.StatusCode != 200 || continueresp.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" {
				break
			} else {
				decoder := eventsource.NewDecoderWithOptions(continueresp.Body, streamOption)
				continueMessage := ""
				for {
					event, err := decoder.Decode()
					if err != nil {
						if err == io.EOF {
							break
						}
						g.Log().Error(ctx, "decoder.Decode error", err)
						break
					}
					text := event.Data()
					if text == "" {
						continue
					}
					if text == "[DONE]" {
						_, err = fmt.Fprintf(rw, "data: %s\n\n", text)
						if err != nil {
							g.Log().Error(ctx, "fmt.Fprintf error", err)
							r.Response.WriteStatusExit(500)
							return
						}
						flusher.Flush()
						continue
					}
					// g.Log().Debug(ctx, "text", gjson.New(text))
					messeage_id := gjson.New(text).Get("message.id").String()
					conversation_id := gjson.New(text).Get("conversation_id").String()
					model_slug := gjson.New(text).Get("message.metadata.model_slug").String()
					finish_type := gjson.New(text).Get("message.metadata.finish_details.type").String()

					// g.Log().Debug(ctx, "conversation_id", conversation_id)
					if conversation_id != "" {
						conversationId = conversation_id
					}
					if model_slug != "" {
						modelSlug = model_slug
					}
					if messeage_id != "" {
						messageId = messeage_id
					}
					if finish_type != "" {
						finishType = finish_type
					}
					// r.Response.Writefln("data: %s\n\n", text)
					// r.Response.Flush()
					textJson := gjson.New(text)
					message := textJson.Get("message.content.parts.0").String()
					if message != "" {
						continueMessage = message
						textJson.Set("message.content.parts.0", messagBody+message)
					}

					_, err = fmt.Fprintf(rw, "data: %s\n\n", textJson)

					if err != nil {
						g.Log().Error(ctx, "fmt.Fprintf error", err)
						continueresp.Body.Close()
						continueresp.Close()
						finishType = "error"
						continue
					}
					flusher.Flush()
				}
				messagBody = messagBody + continueMessage
				g.Log().Debug(ctx, "finishType", finishType)
				g.Log().Debug(ctx, "conversationId", conversationId)
				g.Log().Debug(ctx, "modelSlug", modelSlug)
				g.Log().Debug(ctx, "messageId", messageId)
				// g.Log().Debug(ctx, "messagBody", messagBody)
				continueresp.Body.Close()
				continueresp.Close()

			}

		}
		// 如果请求的会话ID与返回的会话ID不一致，说明是新的会话 需要写入缓存
		if reqJson.Get("conversation_id").String() != conversationId {
			if !history_and_training_disabled {
				cool.CacheManager.Set(ctx, "conversation:"+conversationId, email, time.Hour*24*30)
			}
		}
		r.ExitAll()

		return

	}
	r.Response.WriteStatusExit(resp.StatusCode, resp.ReadAllString())
}

func RefreshSession(email string) {
	ctx := gctx.New()
	m := model.NewChatgptSession()
	result, err := cool.DBM(m).Where("email=?", email).One()
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	g.Log().Info(ctx, "RefreshSession", result["email"], "start")
	// time.Sleep(5 * time.Minute)
	getSessionUrl := config.CHATPROXY(ctx) + "/applelogin"
	var sessionJson *gjson.Json
	refreshToken := gjson.New(result["officialSession"]).Get("refresh_token").String()
	sessionVar := g.Client().SetHeader("authkey", config.AUTHKEY(ctx)).PostVar(ctx, getSessionUrl, g.Map{
		"username":      result["email"],
		"password":      result["password"],
		"refresh_token": refreshToken,
		"authkey":       config.AUTHKEY(ctx),
	})
	sessionJson = gjson.New(sessionVar)
	detail := sessionJson.Get("detail").String()
	if detail == "密码不正确!" || gstr.Contains(detail, "account_deactivated") || gstr.Contains(detail, "403 Forbidden|Unknown or invalid refresh token.") {
		g.Log().Error(ctx, "AddAllSession", email, detail)
		cool.DBM(model.NewChatgptSession()).Where("email=?", email).Update(g.Map{
			"officialSession": sessionJson.String(),
			"status":          0,
		})
		return
	}
	if sessionJson.Get("accessToken").String() == "" {
		g.Log().Error(ctx, "RefreshSession", result["email"], "get session error", sessionJson)
		return
	}
	var isPlus int
	models := sessionJson.Get("models").Array()
	if len(models) > 1 {
		isPlus = 1
	} else {
		isPlus = 0
	}
	_, err = cool.DBM(m).Where("email=?", result["email"]).Update(g.Map{
		"officialSession": sessionJson.String(),
		"status":          1,
		"isPlus":          isPlus,
	})
	if err != nil {
		g.Log().Error(ctx, "RefreshSession", err)
		return
	}
	// 更新缓存
	cacheSession := &config.CacheSession{
		Email:        email,
		AccessToken:  sessionJson.Get("accessToken").String(),
		IsPlus:       isPlus,
		CooldownTime: 0,
	}
	cool.CacheManager.Set(ctx, "session:"+email, cacheSession, time.Hour*24*10)

	// 更新set
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
	g.Log().Info(ctx, "RefreshSession", result["email"], isPlus, "success")

}
