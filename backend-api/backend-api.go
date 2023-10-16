package backendapi

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/modules/chatgpt/service"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/gogf/gf/v2/text/gstr"
)

var (
	ChatgptUserService = service.NewChatgptUserService()
	TraceparentCache   = gcache.New()
)

func init() {
	// 注册路由
	s := g.Server()
	backendApiGroup := s.Group("/backend-api")
	backendApiGroup.POST("/conversation", Conversation)
	backendApiGroup.POST("/files", ProxyAll)
	backendApiGroup.POST("/files/*/uploaded", ProxyAll)

}
func ProxyAll(r *ghttp.Request) {
	ctx := r.GetCtx()
	// g.Dump(r.Request.URL)
	// 获取header中的token Authorization: Bearer xxx 去掉Bearer
	// 获取 Header 中的 Authorization	去除 Bearer
	userToken := r.Header.Get("Authorization")[7:]
	// 如果 Authorization 为空，返回 401
	if userToken == "" {
		r.Response.WriteStatusExit(401)
	}
	Traceparent := r.Header.Get("Traceparent")
	// Traceparent like 00-d8c66cc094b38d1796381c255542f971-09988d8458a2352c-01 获取第二个参数
	// 以-分割，取第二个参数
	TraceparentArr := gstr.Split(Traceparent, "-")
	if len(TraceparentArr) < 2 {
		g.Log().Error(ctx, "Traceparent error", Traceparent)
		r.Response.WriteStatusExit(401)
	}
	// 获取第二个参数
	Traceparent = TraceparentArr[1]
	g.Log().Info(ctx, "Traceparent", Traceparent)

	record, err := cool.DBM(model.NewChatgptUser()).Where("userToken", userToken).Where("isPlus", 1).One()
	if err != nil {
		g.Log().Error(ctx, err)
		r.Response.WriteStatusExit(500)
	}
	if record.IsEmpty() {
		g.Log().Error(ctx, "userToken not found", userToken)
		r.Response.WriteStatusExit(401)
	}
	accessToken := TraceparentCache.MustGet(ctx, Traceparent).String()
	if accessToken == "" {
		sessionPair, code, err := ChatgptUserService.GetSessionPair(ctx, userToken, "", true)
		if err != nil {
			g.Log().Error(ctx, code, err)
			r.Response.WriteStatusExit(code)
			return
		}

		TraceparentCache.Set(ctx, Traceparent, sessionPair.AccessToken, time.Minute)
	}

	UpStream := config.CHATPROXY(ctx)
	u, _ := url.Parse(UpStream)
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		g.Log().Error(ctx, e)
		writer.WriteHeader(http.StatusBadGateway)
	}
	newreq := r.Request.Clone(ctx)
	newreq.URL.Host = u.Host
	newreq.URL.Scheme = u.Scheme
	newreq.Host = u.Host
	newreq.Header.Set("authkey", config.AUTHKEY(ctx))
	newreq.Header.Set("Authorization", "Bearer "+accessToken)
	newreq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	// g.Dump(newreq.URL)
	proxy.ServeHTTP(r.Response.Writer.RawWriter(), newreq)

}
