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
			"error": g.Map{
				"message": "Missing bearer or basic authentication in header",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    nil,
			},
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
				"error": g.Map{
					"message": "Incorrect API key provided: " + userToken + ". You can find your API key at https://platform.openai.com/account/api-keys.",
					"type":    "invalid_request_error",
					"param":   nil,
					"code":    "invalid_api_key",
				},
			})
			return
		}
		if userRecord["isPlus"].Int() == 1 {
			isPlusUser = true
		}
	}

	// g.Log().Debug(ctx, "token: ", token)
	req := &Req{}
	err := r.GetRequestStruct(req)
	if err != nil {
		g.Log().Error(ctx, err)
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "Invalid request",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "invalid_request",
			},
		})
		return
	}
	reqModel := req.Model
	realModel := config.GetModel(ctx, reqModel)
	req.Model = realModel
	isStream := req.Stream
	req.Stream = true
	fullQuestion := ""
	for _, message := range req.Messages {
		fullQuestion += message.Content
	}
	// g.Dump(req)

	// 如果不是plus用户但是使用了plus模型
	if !isPlusUser && gstr.HasPrefix(realModel, "gpt-4") {
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "You are not a plus user, please use the normal model",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "invalid_model",
			},
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
	promptTokens := CountTokens(fullQuestion)

	if isPlusModel {
		if promptTokens > 32*1024 {
			g.Log().Error(ctx, userToken, reqModel, realModel, promptTokens, "tokens too long")
			r.Response.Status = 400
			r.Response.WriteJson(g.Map{
				"error": g.Map{
					"message": "The message you submitted was too long," + gconv.String(promptTokens) + " tokens, please reload the conversation and submit something shorter.",
					"type":    "invalid_request_error",
					"param":   nil,
					"code":    "message_length_exceeds_limit",
				},
			})
			return
		}
	} else {
		if promptTokens > 8*1024 {
			g.Log().Error(ctx, userToken, reqModel, realModel, promptTokens, "tokens too long")
			r.Response.Status = 400
			r.Response.WriteJson(g.Map{
				"error": g.Map{
					"message": "The message you submitted was too long," + gconv.String(promptTokens) + " tokens, please reload the conversation and submit something shorter.",
					"type":    "invalid_request_error",
					"param":   nil,
					"code":    "message_length_exceeds_limit",
				},
			})
			return
		}
	}
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
			"error": g.Map{
				"message": "Server is busy, please try again later",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "server_busy",
			},
		})
		return
	}
	g.Log().Info(ctx, userToken, "使用", emailWithTeamId, reqModel, "->", realModel, "发起会话", "isStream:", isStream)

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
	resp, err := g.Client().SetHeaderMap(reqHeader).Post(ctx, config.CHATPROXY+"/v1/chat/completions", req)
	if err != nil {
		g.Log().Error(ctx, "g.Client().Post error: ", err)
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "Server is busy, please try again later|500",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "server_busy",
			},
		})
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
		// r.Response.WriteStatus(401, resp.ReadAllString())
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "Server is busy, please try again later|" + gconv.String(resp.StatusCode),
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "server_busy",
			},
		})
		return
	}
	if resp.StatusCode == 429 {
		resStr := resp.ReadAllString()
		clears_in = gjson.New(resStr).Get("detail.clears_in").Int()
		g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==429", resStr)
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "Server is busy, please try again later|429",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "server_busy",
			},
		})
		return
	}
	if resp.StatusCode == 413 {
		contentType := resp.Header.Get("Content-Type")
		resStr := resp.ReadAllString()
		g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==413", contentType, realModel, promptTokens, resStr)
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "The message you submitted was too long, please reload the conversation and submit something shorter.",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "message_length_exceeds_limit",
			},
		})
		return
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

		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"error": g.Map{
				"message": "Server is busy, please try again later|403",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    "server_busy",
			},
		})
		return

	}
	if resp.StatusCode == 500 {
		resStr := resp.ReadAllString()
		g.Log().Error(ctx, emailWithTeamId, "resp.StatusCode==500", resStr)
		detail := gjson.New(resStr).Get("detail").String()
		if gstr.Contains(detail, "Sorry! We've encountered an issue with repetitive patterns in your prompt. Please try again with a different prompt.") {
			r.Response.Status = 400
			r.Response.WriteJson(g.Map{
				"error": g.Map{
					"message": "Sorry! We've encountered an issue with repetitive patterns in your prompt. Please try again with a different prompt.",
					"type":    "invalid_request_error",
					"param":   nil,
					"code":    "repetitive_patterns",
				},
			})
			return
		}
	}
	// 如果返回结果不是200
	if resp.StatusCode != 200 {
		g.Log().Error(ctx, "resp.StatusCode: ", resp.StatusCode, resp.ReadAllString())
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
		if isStream {
			r.Response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
			r.Response.Header().Set("Cache-Control", "no-cache")
			r.Response.Header().Set("Connection", "keep-alive")
		} else {
			r.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		modelSlug := ""

		decoder := eventsource.NewDecoder(resp.Body)
		defer decoder.Decode() // 释放资源
		fullMessage := ""

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
				if isStream {
					r.Response.Writeln("data: " + text + "\n")
					r.Response.Flush()
				}
				continue
			}
			respJson := gjson.New(text)
			content := respJson.Get("choices.0.delta.content").String()
			if content != "" {
				fullMessage += content
			}
			replayModel := respJson.Get("model").String()
			if replayModel != "" {
				modelSlug = replayModel
			}
			if isStream {
				respJson.Set("model", reqModel)
				r.Response.Writeln("data: " + respJson.String() + "\n")
				r.Response.Flush()
			}
		}
		if !isStream {
			// g.Log().Info(ctx, fullMessage, CountTokens(fullMessage))
			apiNonStreamResp := gjson.New(ApiRespStr)
			apiNonStreamResp.Set("choices.0.message.content", fullMessage)
			apiNonStreamResp.Set("model", reqModel)
			apiNonStreamResp.Set("id", "chatcmpl-"+gconv.String(time.Now().UnixNano()))
			apiNonStreamResp.Set("created", time.Now().Unix())
			promptTokens := CountTokens(fullQuestion)
			apiNonStreamResp.Set("usage.prompt_tokens", promptTokens)
			completionTokens := CountTokens(fullMessage)
			apiNonStreamResp.Set("usage.completion_tokens", completionTokens)
			apiNonStreamResp.Set("usage.total_tokens", promptTokens+completionTokens)
			r.Response.WriteJson(apiNonStreamResp)
		}

		if realModel != "text-davinci-002-render-sha" && realModel != "auto" && modelSlug == "text-davinci-002-render-sha" {
			isPlusInvalid = true
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "PLUS失效")
		} else {
			g.Log().Info(ctx, userToken, "使用", emailWithTeamId, realModel, "->", modelSlug, "完成会话")
		}
		r.ExitAll()
		return
	}

}
