package backendapi

import (
	"chatgpt-api-server/config"
	"chatgpt-api-server/modules/chatgpt/model"
	"chatgpt-api-server/modules/chatgpt/service"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cool-team-official/cool-admin-go/cool"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

var (
	ChatgptUserService = service.NewChatgptUserService()
)

func init() {
	// 注册路由
	s := g.Server()
	backendApiGroup := s.Group("/backend-api")
	backendApiGroup.POST("/conversation", Conversation)
	backendApiGroup.POST("/files", ProxyAll)
	backendApiGroup.POST("/files/*path", ProxyAll)

}

// NotFound 404
func NotFound(r *ghttp.Request) {
	r.Response.WriteStatus(http.StatusNotFound)
}

func ProxyAll(r *ghttp.Request) {
	ctx := r.GetCtx()
	// 获取header中的token Authorization: Bearer xxx 去掉Bearer
	// 获取 Header 中的 Authorization	去除 Bearer
	userToken := r.Header.Get("Authorization")[7:]
	// 如果 Authorization 为空，返回 401
	if userToken == "" {
		r.Response.WriteStatusExit(401)
	}
	record, err := cool.DBM(model.NewChatgptUser()).Where("userToken", userToken).One()
	if err != nil {
		g.Log().Error(ctx, err)
		r.Response.WriteStatusExit(500)
	}
	if record.IsEmpty() {
		g.Log().Error(ctx, "userToken not found", userToken)
		r.Response.WriteStatusExit(401)
	}
	email := service.SessionQueue.Pop()
	emailStr := gconv.String(email)
	g.Log().Info(ctx, "使用", emailStr, "发起请求")

	defer service.SessionQueue.Push(email)

	accessToken := config.TokenCache.MustGet(ctx, emailStr).String()
	if accessToken == "" {
		g.Log().Error(ctx, "get accessToken from cache fail", emailStr)
		r.Response.WriteStatusExit(401)
		return
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

	// g.Dump(newreq.URL)
	proxy.ServeHTTP(r.Response.Writer.RawWriter(), newreq)

}
