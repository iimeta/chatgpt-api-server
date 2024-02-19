package funcs

import (
	"github.com/gogf/gf/v2/frame/g"
)

type ChatgptRefreshSession struct {
}

func (f *ChatgptRefreshSession) Func(ctx g.Ctx, param string) (err error) {
	g.Log().Info(ctx, "刷新Session ChatgptRefreshSession.Func", "param", param)
	// baseSysLogService := service.NewBaseSysLogService()
	// if param == "true" {
	// 	err = baseSysLogService.Clear(true)
	// } else {
	// 	err = baseSysLogService.Clear(false)
	// }
	return
}

// IsSingleton 上一个任务未执行完成，则跳过本次任务执行
func (f *ChatgptRefreshSession) IsSingleton() bool {
	return true
}

// IsAllWorker 集群模式下是否所有节点有效
func (f *ChatgptRefreshSession) IsAllWorker() bool {
	return false
}

func init() {
	// cool.RegisterFunc("ChatgptRefreshSession", new(ChatgptRefreshSession))
	// g.Dump(cool.FuncMap)
}
