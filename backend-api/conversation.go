package backendapi

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"io"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/launchdarkly/eventsource"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func Conversation(r *ghttp.Request) {
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
	g.Dump(reqJson)

	sessionPair, code, err := ChatgptUserService.GetSessionPair(ctx, userToken, reqJson.Get("conversation_id").String())
	if err != nil {
		r.Response.WriteStatusExit(code)
	}
	// 如果 sessionPair 为空，返回 500
	if sessionPair == nil {
		r.Response.WriteStatusExit(500)
	}

	// 加锁 防止并发
	sessionPair.Lock.Lock()
	// 延迟解锁
	defer func() {
		// 延时1秒
		time.Sleep(time.Second)
		sessionPair.Lock.Unlock()
	}()
	client := g.Client()
	client.SetHeader("Authorization", sessionPair.AccessToken)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("authkey", config.AUTHKEY(ctx))

	resp, err := client.Post(ctx, config.CHATPROXY(ctx)+"/backend-api/conversation", reqJson)
	if err != nil {
		r.Response.WriteStatusExit(500)
	}
	defer resp.Close()
	if resp.StatusCode == 200 && resp.Header.Get("Content-Type") == "text/event-stream; charset=utf-8" {
		conversationId := ""
		decoder := eventsource.NewDecoder(resp.Response.Body)
		for {
			event, err := decoder.Decode()
			if err != nil {
				if err == io.EOF {
					break
				}
				g.Log().Error(ctx, "decoder.Decode error", err)
				continue
			}
			text := event.Data()
			if text == "" {
				continue
			}
			if text == "[DONE]" {
				r.Response.Writefln("data: %s\n\n", text)
				r.Response.Flush()
				break
			}
			conversation_id := gjson.New(text).Get("conversation_id").String()
			// g.Log().Debug(ctx, "conversation_id", conversation_id)
			if conversation_id != "" {
				conversationId = conversation_id
			}
			r.Response.Writefln("data: %s\n\n", text)
			r.Response.Flush()
		}
		// 如果请求的会话ID与返回的会话ID不一致，说明是新的会话，需要插入数据库
		if reqJson.Get("conversation_id").String() != conversationId {
			cool.DBM(model.NewChatgptConversation()).Insert(g.Map{
				"userToken":      userToken,
				"email":          sessionPair.Email,
				"conversationId": conversationId,
			})
		}

		return

	}
	r.Response.WriteStatusExit(resp.StatusCode, resp.ReadAllString())
}
