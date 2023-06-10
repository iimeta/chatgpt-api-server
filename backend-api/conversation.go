package backendapi

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/launchdarkly/eventsource"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gmutex"
)

var (
	USERTOKENLOCKMAP = make(map[string]*gmutex.Mutex)
	continueRequest  = `{"action":"continue","conversation_id":"f8cdda28-fcae-4dc8-b8b6-687af2741ee7","parent_message_id":"c22837bf-c1f9-4579-a2b4-71102670cfe2","model":"text-davinci-002-render-sha","timezone_offset_min":-480,"history_and_training_disabled":false}`
)

func Conversation(r *ghttp.Request) {
	client := g.Client()

	ctx := r.Context()
	// if r.Header.Get("Authorization") == "" {
	// 	r.Response.WriteStatusExit(401)
	// }
	// 获取 Header 中的 Authorization	去除 Bearer
	userToken := r.Header.Get("Authorization")[7:]
	// 如果 Authorization 为空，返回 401
	if userToken == "" {
		r.Response.WriteStatusExit(401)
	}
	reqJson := gjson.New(r.GetBody())
	g.Log().Debug(ctx, userToken, reqJson)
	reqModel := reqJson.Get("model").String()
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
	// if config.PlusModels.ContainsI(reqModel) {
	// 	isPlusModel = true
	// } else if config.FreeModels.Contains(reqModel) {
	// 	isPlusModel = false
	// } else {
	// 	reqJson.Set("model", config.DefaultModel)
	// 	isPlusModel = false
	// }
	// 遍历 config.PlusModels
	g.Log().Debug(ctx, "isPlusModel", isPlusModel)

	sessionPair, code, err := ChatgptUserService.GetSessionPair(ctx, userToken, reqJson.Get("conversation_id").String(), isPlusModel)
	if err != nil {
		g.Log().Error(ctx, code, err)
		r.Response.WriteStatusExit(code)
		return
	}
	// g.Dump(sessionPair)
	// 如果 sessionPair 为空，返回 500
	if sessionPair == nil {
		r.Response.WriteStatusExit(500)
		return
	}
	// 如果配置了  USERTOKENLOCK ,则加锁限制每个用户只能有一个会话并发
	if config.USERTOKENLOCK(ctx) && isPlusModel {
		g.Log().Debug(ctx, "USERTOKENLOCK", config.USERTOKENLOCK(ctx))
		// 如果 USERTOKENLOCKMAP 中没有这个用户的锁，则创建一个
		if _, ok := USERTOKENLOCKMAP[userToken]; !ok {
			USERTOKENLOCKMAP[userToken] = gmutex.New()
		}
		// 加锁
		if USERTOKENLOCKMAP[userToken].TryLock() {
			g.Log().Debug(ctx, userToken, "加锁USERTOKENLOCK")
			// 延迟解锁
			defer func() {
				// 延时1秒
				time.Sleep(time.Second)
				USERTOKENLOCKMAP[userToken].Unlock()
				g.Log().Debug(ctx, userToken, "解锁USERTOKENLOCK")
			}()
		} else {
			g.Log().Info(ctx, userToken, "触发USERTOKENLOCK,返回429")
			r.Response.WriteStatusExit(429)
			return
		}
	}

	// 加锁 防止并发
	sessionPair.Lock.Lock()
	g.Log().Debug(ctx, userToken, "加锁sessionPair.Lock")
	// 延迟解锁
	defer func() {
		// 延时1秒
		time.Sleep(time.Second)
		sessionPair.Lock.Unlock()
		g.Log().Debug(ctx, userToken, "解锁sessionPair.Lock")
	}()
	// client := g.Client()
	client.SetHeader("Authorization", sessionPair.AccessToken)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("authkey", config.AUTHKEY(ctx))

	resp, err := client.Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", reqJson)
	if err != nil {
		r.Response.WriteStatusExit(500)
	}
	defer resp.Close()
	defer resp.Body.Close()

	if resp.StatusCode == 200 && resp.Header.Get("Content-Type") == "text/event-stream; charset=utf-8" {
		r.Response.Header().Set("Content-Type", "text/event-stream")
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
			message_body := gjson.New(text).Get("message.content.parts.0").String()

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
		// g.Log().Debug(ctx, "messagBody", messagBody)
		// 如果是max_tokens类型的完成,说明会话未结束，需要继续请求
		count := 0
		for finishType == "max_tokens" && count < 3 {
			g.Log().Debug(ctx, "finishType", finishType, "继续请求")
			continueJson := gjson.New(continueRequest)
			continueJson.Set("conversation_id", conversationId)
			continueJson.Set("model", modelSlug)
			continueJson.Set("parent_message_id", messageId)
			g.Log().Debug(ctx, "continueJson", continueJson)
			continueresp, err := client.Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", continueJson)
			if err != nil {
				r.Response.WriteStatusExit(500)
			}
			defer continueresp.Close()
			defer continueresp.Body.Close()
			g.Log().Debug(ctx, "continueresp.StatusCode", continueresp.StatusCode)
			if continueresp.StatusCode == 200 && continueresp.Header.Get("Content-Type") == "text/event-stream; charset=utf-8" {
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
				count++
				g.Log().Debug(ctx, "finishType", finishType)
				g.Log().Debug(ctx, "conversationId", conversationId)
				g.Log().Debug(ctx, "modelSlug", modelSlug)
				g.Log().Debug(ctx, "messageId", messageId)
				// g.Log().Debug(ctx, "messagBody", messagBody)
				continueresp.Body.Close()
				continueresp.Close()

			}

		}
		// 如果请求的会话ID与返回的会话ID不一致，说明是新的会话，需要插入数据库
		if reqJson.Get("conversation_id").String() != conversationId {
			cool.DBM(model.NewChatgptConversation()).Insert(g.Map{
				"userToken":      userToken,
				"email":          sessionPair.Email,
				"conversationId": conversationId,
			})
		}
		r.ExitAll()

		return

	}
	r.Response.WriteStatusExit(resp.StatusCode, resp.ReadAllString())
}
