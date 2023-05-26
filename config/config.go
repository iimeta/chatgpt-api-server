package config

import "github.com/gogf/gf/v2/frame/g"

func CHATPROXY(ctx g.Ctx) string {
	return g.Cfg().MustGetWithEnv(ctx, "CHATPROXY").String()
}
