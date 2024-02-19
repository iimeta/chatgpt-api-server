package chat

import (
	"chatgpt-api-server/apireq"
	"chatgpt-api-server/apirespstream"
	backendapi "chatgpt-api-server/backend-api"
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/utility"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/launchdarkly/eventsource"
)

var (
	// client    = g.Client()
	ErrNoAuth = `{
		"error": {
			"message": "You didn't provide an API key. You need to provide your API key in an Authorization header using Bearer auth (i.e. Authorization: Bearer YOUR_KEY), or as the password field (with blank username) if you're accessing the API from your browser and are prompted for a username and password. You can obtain an API key from https://platform.openai.com/account/api-keys.",
			"type": "invalid_request_error",
			"param": null,
			"code": null
		}
	}`
	ErrKeyInvalid = `{
		"error": {
			"message": "Incorrect API key provided: sk-4yNZz***************************************6mjw. You can find your API key at https://platform.openai.com/account/api-keys.",
			"type": "invalid_request_error",
			"param": null,
			"code": "invalid_api_key"
		}
	}`
	ChatReqStr = `{
		"action": "next",
		"messages": [
		  {
			"id": "aaa2f210-64e1-4f0d-aa51-e73fe1ae74af",
			"author": { "role": "user" },
			"content": { "content_type": "text", "parts": ["1\n"] },
			"metadata": {}
		  }
		],
		"parent_message_id": "aaa1a8ab-61d6-4fc0-a5f5-181015c2ebaf",
		"model": "text-davinci-002-render-sha",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": { "kind": "primary_assistant" }
	  }
	  `
	ChatTurboReqStr = `
	{
		"action": "next",
		"messages": [
			{
				"id": "aaa2b2cc-e7e9-47c5-8171-0ff8a6d9d6d3",
				"author": {
					"role": "user"
				},
				"content": {
					"content_type": "text",
					"parts": [
						"你好"
					]
				},
				"metadata": {}
			}
		],
		"parent_message_id": "aaa1403d-c61e-4818-90e0-93a99465aec6",
		"model": "gpt-4",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": {
			"kind": "primary_assistant"
		}
	}`
	Chat4ReqStr = `
	{
		"action": "next",
		"messages": [
			{
				"id": "aaa2b182-d834-4f30-91f3-f791fa953204",
				"author": {
					"role": "user"
				},
				"content": {
					"content_type": "text",
					"parts": [
						"画一只猫1231231231"
					]
				},
				"metadata": {}
			}
		],
		"parent_message_id": "aaa11581-bceb-46c5-bc76-cb84be69725e",
		"model": "gpt-4-gizmo",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": {
			"gizmo": {
				"gizmo": {
					"id": "g-YyyyMT9XH",
					"organization_id": "org-OROoM5KiDq6bcfid37dQx4z4",
					"short_url": "g-YyyyMT9XH-chatgpt-classic",
					"author": {
						"user_id": "user-u7SVk5APwT622QC7DPe41GHJ",
						"display_name": "ChatGPT",
						"selected_display": "name",
						"is_verified": true
					},
					"voice": {
						"id": "ember"
					},
					"display": {
						"name": "ChatGPT Classic",
						"description": "The latest version of GPT-4 with no additional capabilities",
						"welcome_message": "Hello",
						"profile_picture_url": "https://files.oaiusercontent.com/file-i9IUxiJyRubSIOooY5XyfcmP?se=2123-10-13T01%3A11%3A31Z&sp=r&sv=2021-08-06&sr=b&rscc=max-age%3D31536000%2C%20immutable&rscd=attachment%3B%20filename%3Dgpt-4.jpg&sig=ZZP%2B7IWlgVpHrIdhD1C9wZqIvEPkTLfMIjx4PFezhfE%3D",
						"categories": []
					},
					"share_recipient": "link",
					"updated_at": "2023-11-06T01:11:32.191060+00:00",
					"last_interacted_at": "2023-11-18T07:50:19.340421+00:00",
					"tags": [
						"public",
						"first_party"
					]
				},
				"tools": [],
				"files": [],
				"product_features": {
					"attachments": {
						"type": "retrieval",
						"accepted_mime_types": [
							"text/x-c",
							"text/html",
							"application/x-latext",
							"text/plain",
							"text/x-ruby",
							"text/x-typescript",
							"text/x-c++",
							"text/x-java",
							"text/x-sh",
							"application/vnd.openxmlformats-officedocument.presentationml.presentation",
							"text/x-script.python",
							"text/javascript",
							"text/x-tex",
							"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
							"application/msword",
							"application/pdf",
							"text/x-php",
							"text/markdown",
							"application/json",
							"text/x-csharp"
						],
						"image_mime_types": [
							"image/jpeg",
							"image/png",
							"image/gif",
							"image/webp"
						],
						"can_accept_all_mime_types": true
					}
				}
			},
			"kind": "gizmo_interaction",
			"gizmo_id": "g-YyyyMT9XH"
		},
		"force_paragen": false,
		"force_rate_limit": false
	}`
	ApiRespStr = `{
		"id": "chatcmpl-LLKfuOEHqVW2AtHks7wAekyrnPAoj",
		"object": "chat.completion",
		"created": 1689864805,
		"model": "gpt-3.5-turbo",
		"usage": {
			"prompt_tokens": 0,
			"completion_tokens": 0,
			"total_tokens": 0
		},
		"choices": [
			{
				"message": {
					"role": "assistant",
					"content": "Hello! How can I assist you today?"
				},
				"finish_reason": "stop",
				"index": 0
			}
		]
	}`
	ApiRespStrStream = `{
		"id": "chatcmpl-afUFyvbTa7259yNeDqaHRBQxH2PLH",
		"object": "chat.completion.chunk",
		"created": 1689867370,
		"model": "gpt-3.5-turbo",
		"choices": [
			{
				"delta": {
					"content": "Hello"
				},
				"index": 0,
				"finish_reason": null
			}
		]
	}`
	ApiRespStrStreamEnd = `{"id":"apirespid","object":"chat.completion.chunk","created":apicreated,"model": "apirespmodel","choices":[{"delta": {},"index": 0,"finish_reason": "stop"}]}`
)

func Completions(r *ghttp.Request) {
	ctx := r.Context()
	// g.Log().Debug(ctx, "Conversation start....................")
	if config.MAXTIME > 0 {
		// 创建带有超时的context
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.MAXTIME)*time.Second)
		defer cancel()
	}

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
	// 从请求中获取参数
	req := &apireq.Req{}
	err := r.GetRequestStruct(req)
	if err != nil {
		g.Log().Error(ctx, "r.GetRequestStruct(req) error: ", err)
		r.Response.Status = 400
		r.Response.WriteJson(gjson.New(`{"error": "bad request"}`))
		return
	}
	// g.Dump(req)
	// 遍历 req.Messages 拼接 newMessages
	newMessages := ""
	for _, message := range req.Messages {
		newMessages += message.Content + "\n"
	}
	// g.Dump(newMessages)
	var ChatReq *gjson.Json
	if gstr.HasPrefix(req.Model, "gpt-4") {
		ChatReq = gjson.New(Chat4ReqStr)
	} else {
		ChatReq = gjson.New(ChatReqStr)
	}

	if gstr.HasPrefix(req.Model, "gpt-4-turbo") {
		ChatReq = gjson.New(ChatTurboReqStr)
	}

	if req.Model == "gpt-4-mobile" {
		ChatReq = gjson.New(ChatReqStr)
		ChatReq.Set("model", "gpt-4-mobile")
	}

	// 如果不是plus用户但是使用了plus模型
	if !isPlusUser && gstr.HasPrefix(req.Model, "gpt-4") {
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
	isPlusModel := gstr.HasPrefix(req.Model, "gpt-4")
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
						return
					}
					if clears_in > 0 {
						// 延迟归还
						time.Sleep(time.Duration(clears_in) * time.Second)
					}
					config.PlusSet.Add(emailWithTeamId)
				}
			}()
		}()
		if email == "" {
			emailWithTeamId, ok = config.PlusSet.Pop()
			g.Log().Info(ctx, emailWithTeamId, ok)
			if !ok {
				g.Log().Error(ctx, "Get email from set error")
				r.Response.Status = 500
				r.Response.WriteJson(g.Map{
					"detail": "Server is busy, please try again later",
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
	realModel := ChatReq.Get("model").String()
	g.Log().Info(ctx, userToken, "使用", emailWithTeamId, req.Model, "->", realModel, "发起会话")

	// 使用email获取 accessToken
	sessionCache := &config.CacheSession{}
	cool.CacheManager.MustGet(ctx, "session:"+email).Scan(&sessionCache)
	accessToken := sessionCache.AccessToken
	err = utility.CheckAccessToken(accessToken)
	if err != nil { // accessToken失效
		g.Log().Error(ctx, err)
		isReturn = false
		cool.DBM(model.NewChatgptSession()).Where("email", email).Update(g.Map{"status": 0})

		go backendapi.RefreshSession(email)
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"detail": "accessToken is invalid,will be refresh",
		})
		return
	}
	ChatReq.Set("messages.0.content.parts.0", newMessages)
	ChatReq.Set("messages.0.id", uuid.NewString())
	ChatReq.Set("parent_message_id", uuid.NewString())
	if len(req.PluginIds) > 0 {
		ChatReq.Set("plugin_ids", req.PluginIds)
	}
	ChatReq.Set("history_and_training_disabled", true)

	// ChatReq.Dump()
	// 请求openai
	reqHeader := g.MapStrStr{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
		"authkey":       config.AUTHKEY(ctx),
	}
	if teamId != "" {
		reqHeader["ChatGPT-Account-ID"] = teamId
	}
	resp, err := g.Client().SetHeaderMap(reqHeader).Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", ChatReq.MustToJson())
	if err != nil {
		g.Log().Error(ctx, "g.Client().Post error: ", err)
		r.Response.Status = 500
		r.Response.WriteJson(gjson.New(`{"detail": "internal server error"}`))
		return
	}
	defer resp.Close()
	if resp.StatusCode == 401 {
		g.Log().Error(ctx, "token过期,需要重新获取token", email, resp.ReadAllString())
		isReturn = false
		cool.DBM(model.NewChatgptSession()).Where("email", email).Update(g.Map{"status": 0})
		go backendapi.RefreshSession(email)
		r.Response.WriteStatus(401, resp.ReadAllString())
		return
	}
	if resp.StatusCode == 429 {
		resStr := resp.ReadAllString()

		clears_in = gjson.New(resStr).Get("detail.clears_in").Int()

		if clears_in > 0 {
			g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==430", resStr)

			r.Response.WriteStatusExit(430, resStr)
			return
		} else {
			g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==429", resStr)

			r.Response.WriteStatusExit(429, resStr)
			return
		}
	}
	// 如果返回结果不是200
	if resp.StatusCode != 200 {
		g.Log().Error(ctx, "resp.StatusCode: ", resp.StatusCode)
		r.Response.Status = resp.StatusCode
		r.Response.WriteJson(gjson.New(resp.ReadAllString()))
		return
	}
	// if resp.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" && resp.Header.Get("Content-Type") != "text/event-stream" {
	// 	g.Log().Error(ctx, "resp.Header.Get(Content-Type): ", resp.Header.Get("Content-Type"))
	// 	r.Response.Status = 500
	// 	r.Response.WriteJson(gjson.New(`{"detail": "internal server error"}`))
	// 	return
	// }

	// 流式返回
	if req.Stream {
		r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		r.Response.Header().Set("Cache-Control", "no-cache")
		r.Response.Header().Set("Connection", "keep-alive")
		modelSlug := ""

		message := ""
		decoder := eventsource.NewDecoder(resp.Body)
		defer decoder.Decode()

		id := config.GenerateID(29)
		for {
			event, err := decoder.Decode()
			if err != nil {
				// if err == io.EOF {
				// 	break
				// }
				// g.Log().Info(ctx, "释放资源")
				break
			}
			text := event.Data()
			// g.Log().Debug(ctx, "text: ", text)
			if text == "" {
				continue
			}
			if text == "[DONE]" {
				apiRespStrEnd := gstr.Replace(ApiRespStrStreamEnd, "apirespid", id)
				apiRespStrEnd = gstr.Replace(apiRespStrEnd, "apicreated", gconv.String(time.Now().Unix()))
				apiRespStrEnd = gstr.Replace(apiRespStrEnd, "apirespmodel", req.Model)
				r.Response.Writefln("data: " + apiRespStrEnd + "\n\n")
				r.Response.Flush()
				r.Response.Writefln("data: " + text + "\n\n")
				r.Response.Flush()
				continue
				// resp.Close()

				// break
			}
			// gjson.New(text).Dump()
			role := gjson.New(text).Get("message.author.role").String()
			if role == "assistant" {
				messageTemp := gjson.New(text).Get("message.content.parts.0").String()
				model_slug := gjson.New(text).Get("message.metadata.model_slug").String()
				if model_slug != "" {
					modelSlug = model_slug
				}
				// g.Log().Debug(ctx, "messageTemp: ", messageTemp)
				// 如果 messageTemp 不包含 message 且plugin_ids为空
				if !gstr.Contains(messageTemp, message) && len(req.PluginIds) == 0 {
					continue
				}

				content := strings.Replace(messageTemp, message, "", 1)
				if content == "" {
					continue
				}
				message = messageTemp
				apiResp := gjson.New(ApiRespStrStream)
				apiResp.Set("id", id)
				apiResp.Set("created", time.Now().Unix())
				apiResp.Set("choices.0.delta.content", content)
				// if req.Model == "gpt-4" {
				// 	apiResp.Set("model", "gpt-4")
				// }
				apiResp.Set("model", req.Model)
				apiRespStruct := &apirespstream.ApiRespStreamStruct{}
				gconv.Struct(apiResp, apiRespStruct)
				// g.Dump(apiRespStruct)
				// 创建一个jsoniter的Encoder
				json := jsoniter.ConfigCompatibleWithStandardLibrary

				// 将结构体转换为JSON文本并保持顺序
				sortJson, err := json.Marshal(apiRespStruct)
				if err != nil {
					fmt.Println("转换JSON出错:", err)
					continue
				}
				r.Response.Writeln("data: " + string(sortJson) + "\n\n")
				r.Response.Flush()
			}

		}

		if realModel != "text-davinci-002-render-sha" && modelSlug == "text-davinci-002-render-sha" {
			isPlusInvalid = true
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}

	} else {
		// 非流式回应
		modelSlug := ""
		content := ""
		decoder := eventsource.NewDecoder(resp.Body)
		defer decoder.Decode()

		for {
			event, err := decoder.Decode()
			if err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			text := event.Data()
			if text == "" {
				continue
			}
			if text == "[DONE]" {
				resp.Close()
				break
			}
			// gjson.New(text).Dump()
			role := gjson.New(text).Get("message.author.role").String()
			if role == "assistant" {
				model_slug := gjson.New(text).Get("message.metadata.model_slug").String()
				if model_slug != "" {
					modelSlug = model_slug
				}
				message := gjson.New(text).Get("message.content.parts.0").String()
				if message != "" {
					content = message
				}
			}
		}
		completionTokens := CountTokens(content)
		promptTokens := CountTokens(newMessages)
		totalTokens := completionTokens + promptTokens

		apiResp := gjson.New(ApiRespStr)
		apiResp.Set("choices.0.message.content", content)
		id := config.GenerateID(29)
		apiResp.Set("id", id)
		apiResp.Set("created", time.Now().Unix())
		// if req.Model == "gpt-4" {
		// 	apiResp.Set("model", "gpt-4")
		// }
		apiResp.Set("model", req.Model)

		apiResp.Set("usage.prompt_tokens", promptTokens)
		apiResp.Set("usage.completion_tokens", completionTokens)
		apiResp.Set("usage.total_tokens", totalTokens)
		r.Response.WriteJson(apiResp)
		if realModel != "text-davinci-002-render-sha" && modelSlug == "text-davinci-002-render-sha" {
			isPlusInvalid = true

			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}
	}

}
